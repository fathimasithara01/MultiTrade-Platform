package service

import (
	"context"
	"fmt"
)

type NotificationService interface {
	SendNotification(ctx context.Context, userID int64, message string) error
}

type notificationService struct{}

func NewNotificationService() NotificationService {
	return &notificationService{}
}

func (s *notificationService) SendNotification(ctx context.Context, userID int64, message string) error {
	fmt.Printf("📢 NOTIFICATION [User #%d]: %s\n", userID, message)
	return nil
}
