package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/asset"
	"github.com/fathimasithara01/multitrade-platform/internal/asset/dto"
	"github.com/fathimasithara01/multitrade-platform/internal/asset/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/redis"
)

var (
	ErrNotAssetOwner    = errors.New("you do not own this asset")
	ErrDuplicateSymbol  = errors.New("an asset with this symbol already exists")
	ErrInvalidAssetData = errors.New("invalid asset data")
)

type AssetService struct {
	assetRepo repository.AssetRepository
	cache     *redis.Client
}

func NewAssetService(assetRepo repository.AssetRepository, cache *redis.Client) *AssetService {
	return &AssetService{assetRepo: assetRepo, cache: cache}
}

func (s *AssetService) CreateAsset(ctx context.Context, brokerID int64, input dto.CreateAssetInput) (*asset.Asset, error) {
	if err := validatePositiveDecimal(input.Price); err != nil {
		return nil, fmt.Errorf("%w: price must be > 0", ErrInvalidAssetData)
	}
	if err := validatePositiveDecimal(input.Quantity); err != nil {
		return nil, fmt.Errorf("%w: quantity must be > 0", ErrInvalidAssetData)
	}

	a := &asset.Asset{
		BrokerID:    brokerID,
		Symbol:      input.Symbol,
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Quantity:    input.Quantity,
		Status:      asset.AssetStatusActive,
	}

	created, err := s.assetRepo.Create(ctx, a)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateSymbol
		}
		return nil, fmt.Errorf("create asset: %w", err)
	}

	s.cache.SetPrice(ctx, created.ID, created.Price)

	log.Info().Int64("broker_id", brokerID).Str("symbol", created.Symbol).Msg("Asset created")
	return created, nil
}

func (s *AssetService) UpdateAsset(ctx context.Context, brokerID, assetID int64, input dto.UpdateAssetInput) (*asset.Asset, error) {
	existing, err := s.assetRepo.GetByID(ctx, assetID)
	if errors.Is(err, repository.ErrAssetNotFound) {
		return nil, repository.ErrAssetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update asset: get: %w", err)
	}

	if existing.BrokerID != brokerID {
		return nil, ErrNotAssetOwner
	}

	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Description != nil {
		existing.Description = input.Description
	}
	if input.Price != nil {
		if err := validatePositiveDecimal(*input.Price); err != nil {
			return nil, fmt.Errorf("%w: price must be > 0", ErrInvalidAssetData)
		}
		existing.Price = *input.Price
	}
	if input.Quantity != nil {
		if err := validatePositiveDecimal(*input.Quantity); err != nil {
			return nil, fmt.Errorf("%w: quantity must be > 0", ErrInvalidAssetData)
		}
		existing.Quantity = *input.Quantity
	}

	updated, err := s.assetRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("update asset: save: %w", err)
	}

	s.cache.InvalidatePrice(ctx, updated.ID)
	s.cache.SetPrice(ctx, updated.ID, updated.Price)

	log.Info().Int64("broker_id", brokerID).Int64("asset_id", assetID).Msg("Asset updated")
	return updated, nil
}

func (s *AssetService) DisableAsset(ctx context.Context, brokerID, assetID int64) (*asset.Asset, error) {
	existing, err := s.assetRepo.GetByID(ctx, assetID)
	if errors.Is(err, repository.ErrAssetNotFound) {
		return nil, repository.ErrAssetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("disable asset: get: %w", err)
	}

	if existing.BrokerID != brokerID {
		return nil, ErrNotAssetOwner
	}

	disabled := *existing
	disabled.Status = asset.AssetStatusDisabled

	updated, err := s.assetRepo.Update(ctx, &disabled)
	if err != nil {
		return nil, fmt.Errorf("disable asset: save: %w", err)
	}

	s.cache.InvalidatePrice(ctx, updated.ID)

	log.Info().Int64("broker_id", brokerID).Int64("asset_id", assetID).Msg("Asset disabled")
	return updated, nil
}

func (s *AssetService) GetAsset(ctx context.Context, id int64) (*asset.Asset, error) {
	a, err := s.assetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if cached, ok := s.cache.GetPrice(ctx, id); ok {
		a.Price = cached
	} else {
		s.cache.SetPrice(ctx, id, a.Price)
	}

	return a, nil
}

func (s *AssetService) ListActiveAssets(ctx context.Context) ([]asset.Asset, error) {
	assets, err := s.assetRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	for i, a := range assets {
		if cached, ok := s.cache.GetPrice(ctx, a.ID); ok {
			assets[i].Price = cached
		} else {
			s.cache.SetPrice(ctx, a.ID, a.Price)
		}
	}

	return assets, nil
}

func validatePositiveDecimal(s string) error {
	if s == "" {
		return ErrInvalidAssetData
	}
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err != nil || val <= 0 {
		return ErrInvalidAssetData
	}
	return nil
}

func isDuplicateKeyError(err error) bool {
	return err != nil && (contains(err.Error(), "23505") || contains(err.Error(), "unique"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
