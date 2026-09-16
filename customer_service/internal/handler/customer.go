package handler

import (
	customerv1 "github.com/mikhaeris/sky-bank/customer_service/api/customer/v1"
	"github.com/mikhaeris/sky-bank/customer_service/internal/service"
)

type CustomerHandler struct {
	customerv1.UnimplementedCustomerServer
	customerv1.UnimplementedCustomerPrivateServer
}

func NewCustomerHandler(customerService *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{}
}
