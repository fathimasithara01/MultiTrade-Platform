package service

import (
	"context"
)

type ComplianceService interface {
	CheckUser(ctx context.Context, userID int64) (bool, string, float64, error)
}

type complianceService struct{}

func NewComplianceService() ComplianceService {
	return &complianceService{}
}

func (s *complianceService) CheckUser(ctx context.Context, userID int64) (bool, string, float64, error) {
	return true, "User KYC is active and compliant", 10000.0, nil
}
