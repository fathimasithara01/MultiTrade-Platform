package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/kafka"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet/dto"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet/repository"
)

var ErrInvalidAmount = errors.New("amount must be greater than zero")

type WalletService struct {
	db            *sqlx.DB
	walletRepo    repository.WalletRepository
	kafkaProducer *kafka.Producer
}

func NewWalletService(db *sqlx.DB, walletRepo repository.WalletRepository, kafkaProducer *kafka.Producer) *WalletService {
	return &WalletService{db: db, walletRepo: walletRepo, kafkaProducer: kafkaProducer}
}

func (s *WalletService) CreateWalletForUser(ctx context.Context, userID int64) error {
	_, err := s.walletRepo.CreateWallet(ctx, nil, userID)
	return err
}

func (s *WalletService) GetWallet(ctx context.Context, userID int64) (*wallet.Wallet, error) {
	w, err := s.walletRepo.GetByUserID(ctx, userID)
	if errors.Is(err, repository.ErrWalletNotFound) {
		return nil, repository.ErrWalletNotFound
	}
	return w, err
}

func (s *WalletService) Deposit(ctx context.Context, userID int64, input dto.AmountInput) (*wallet.Wallet, *wallet.WalletTransaction, error) {
	amount, err := parsePositiveAmount(input.Amount)
	if err != nil {
		return nil, nil, err
	}

	var updatedWallet *wallet.Wallet
	var txRecord *wallet.WalletTransaction

	err = s.runTx(ctx, func(tx *sqlx.Tx) error {
		w, err := s.walletRepo.GetByUserIDForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}

		current := mustParseBig(w.Balance)
		newBal := new(big.Float).Add(current, amount)
		newBalStr := formatAmount(newBal)

		if err := s.walletRepo.UpdateBalance(ctx, tx, w.ID, newBalStr, w.Version); err != nil {
			return err
		}

		wtx := &wallet.WalletTransaction{
			WalletID:        w.ID,
			Amount:          formatAmount(amount),
			TransactionType: wallet.TxTypeDeposit,
			Status:          wallet.TxStatusCompleted,
			Description:     input.Description,
		}
		txRecord, err = s.walletRepo.InsertTransaction(ctx, tx, wtx)
		if err != nil {
			return err
		}

		updatedWallet = &wallet.Wallet{
			ID:       w.ID,
			UserID:   w.UserID,
			Balance:  newBalStr,
			Currency: w.Currency,
			Version:  w.Version + 1,
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	log.Info().Int64("user_id", userID).Str("amount", formatAmount(amount)).Msg("Deposit completed")

	if s.kafkaProducer != nil {
		eventPayload := kafka.WalletEventPayload{
			WalletID:        updatedWallet.ID,
			UserID:          updatedWallet.UserID,
			Balance:         updatedWallet.Balance,
			ChangeAmount:    formatAmount(amount),
			TransactionType: "deposit",
			Timestamp:       time.Now().UTC(),
		}
		_ = s.kafkaProducer.PublishEvent(ctx, "wallet-events", "wallet.updated", eventPayload)
	}

	return updatedWallet, txRecord, nil
}

func (s *WalletService) Withdraw(ctx context.Context, userID int64, input dto.AmountInput) (*wallet.Wallet, *wallet.WalletTransaction, error) {
	amount, err := parsePositiveAmount(input.Amount)
	if err != nil {
		return nil, nil, err
	}

	var updatedWallet *wallet.Wallet
	var txRecord *wallet.WalletTransaction

	err = s.runTx(ctx, func(tx *sqlx.Tx) error {
		w, err := s.walletRepo.GetByUserIDForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}

		current := mustParseBig(w.Balance)

		if current.Cmp(amount) < 0 {
			return repository.ErrInsufficientBalance
		}

		newBal := new(big.Float).Sub(current, amount)
		newBalStr := formatAmount(newBal)

		if err := s.walletRepo.UpdateBalance(ctx, tx, w.ID, newBalStr, w.Version); err != nil {
			return err
		}

		wtx := &wallet.WalletTransaction{
			WalletID:        w.ID,
			Amount:          formatAmount(amount),
			TransactionType: wallet.TxTypeWithdrawal,
			Status:          wallet.TxStatusCompleted,
			Description:     input.Description,
		}
		txRecord, err = s.walletRepo.InsertTransaction(ctx, tx, wtx)
		if err != nil {
			return err
		}

		updatedWallet = &wallet.Wallet{
			ID:       w.ID,
			UserID:   w.UserID,
			Balance:  newBalStr,
			Currency: w.Currency,
			Version:  w.Version + 1,
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	log.Info().Int64("user_id", userID).Str("amount", formatAmount(amount)).Msg("Withdrawal completed")

	if s.kafkaProducer != nil {
		eventPayload := kafka.WalletEventPayload{
			WalletID:        updatedWallet.ID,
			UserID:          updatedWallet.UserID,
			Balance:         updatedWallet.Balance,
			ChangeAmount:    formatAmount(amount),
			TransactionType: "withdrawal",
			Timestamp:       time.Now().UTC(),
		}
		_ = s.kafkaProducer.PublishEvent(ctx, "wallet-events", "wallet.updated", eventPayload)
	}

	return updatedWallet, txRecord, nil
}

func (s *WalletService) ListTransactions(ctx context.Context, userID int64, page, pageSize int) (*dto.TransactionPage, error) {
	w, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	txs, err := s.walletRepo.ListTransactions(ctx, w.ID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.walletRepo.CountTransactions(ctx, w.ID)
	if err != nil {
		return nil, err
	}

	// Convert transactions to DTOs
	txDTOs := make([]dto.WalletTransactionDTO, len(txs))
	for i, tx := range txs {
		txDTOs[i] = dto.WalletTransactionDTO{
			ID:              tx.ID,
			WalletID:        tx.WalletID,
			Amount:          tx.Amount,
			TransactionType: tx.TransactionType,
			Status:          tx.Status,
			ReferenceID:     tx.ReferenceID,
			Description:     tx.Description,
			CreatedAt:       tx.CreatedAt,
		}
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return &dto.TransactionPage{
		Data:       txDTOs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *WalletService) runTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func parsePositiveAmount(s string) (*big.Float, error) {
	f, ok := new(big.Float).SetPrec(64).SetString(s)
	if !ok {
		return nil, ErrInvalidAmount
	}
	if f.Cmp(big.NewFloat(0)) <= 0 {
		return nil, ErrInvalidAmount
	}
	return f, nil
}

func mustParseBig(s string) *big.Float {
	f, _ := new(big.Float).SetPrec(64).SetString(s)
	return f
}

func formatAmount(f *big.Float) string {
	return fmt.Sprintf("%.8f", f)
}
