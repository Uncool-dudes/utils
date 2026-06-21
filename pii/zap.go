package pii

import "go.uber.org/zap"

// EmailField returns a zap field that logs the masked email value.
func EmailField(key string, v Email) zap.Field { return zap.String(key, v.Masked()) }

// PhoneField returns a zap field that logs the masked phone value.
func PhoneField(key string, v Phone) zap.Field { return zap.String(key, v.Masked()) }

// IPField returns a zap field that logs the masked IP address value.
func IPField(key string, v IPAddress) zap.Field { return zap.String(key, v.Masked()) }

// FirstNameField returns a zap field that logs the masked first name value.
func FirstNameField(key string, v FirstName) zap.Field { return zap.String(key, v.Masked()) }

// LastNameField returns a zap field that logs the masked last name value.
func LastNameField(key string, v LastName) zap.Field { return zap.String(key, v.Masked()) }

// FullNameField returns a zap field that logs the masked full name value.
func FullNameField(key string, v FullName) zap.Field { return zap.String(key, v.Masked()) }

// TaxIDField returns a zap field that logs the masked tax ID value.
func TaxIDField(key string, v TaxID) zap.Field { return zap.String(key, v.Masked()) }
