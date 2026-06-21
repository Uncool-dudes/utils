package pii

import "go.uber.org/zap"

// Masked zap field constructors — emit the masked value, never plaintext.
// Use instead of zap.String for any PII field.
//
//	log.Info("user created", pii.EmailField("email", user.Email))
//	→ {"email": "j***@example.com"}
func EmailField(key string, v Email) zap.Field         { return zap.String(key, v.Masked()) }
func PhoneField(key string, v Phone) zap.Field         { return zap.String(key, v.Masked()) }
func IPField(key string, v IPAddress) zap.Field        { return zap.String(key, v.Masked()) }
func FirstNameField(key string, v FirstName) zap.Field { return zap.String(key, v.Masked()) }
func LastNameField(key string, v LastName) zap.Field   { return zap.String(key, v.Masked()) }
func FullNameField(key string, v FullName) zap.Field   { return zap.String(key, v.Masked()) }
func TaxIDField(key string, v TaxID) zap.Field         { return zap.String(key, v.Masked()) }
