package service

import (
	"github.com/mikhaeris/sky-bank/customer_service/internal/repository"
	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
)

type CustomerService struct {
	customerRepo       *repository.CustomerRepository
	notificationClient notificationv1.NotificationServiceClient
}

func NewCustomerService(
	customerRepo *repository.CustomerRepository,
	notificationClient notificationv1.NotificationServiceClient,
) *CustomerService {
	return &CustomerService{
		customerRepo:       customerRepo,
		notificationClient: notificationClient,
	}
}
