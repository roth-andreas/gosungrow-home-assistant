package iSolarCloud

import "time"

//goland:noinspection SpellCheckingInspection
const (
	DefaultHost      = "https://augateway.isolarcloud.com"
	DefaultApiAppKey = "B0455FBE7AA0328DB57B59AA729F05D8"
	DefaultAccessKey = "9grzgbmxdsp3arfmmgq347xjbza4ysps"
	// URL-safe base64 encoded RSA public key used for x-random-secret-key / x-limit-obj headers.
	DefaultEncryptPublicKey = "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCkecphb6vgsBx4LJknKKes-eyj7-RKQ3fikF5B67EObZ3t4moFZyMGuuJPiadYdaxvRqtxyblIlVM7omAasROtKRhtgKwwRxo2a6878qBhTgUVlsqugpI_7ZC9RmO2Rpmr8WzDeAapGANfHN5bVr7G7GYGwIrjvyxMrAVit_oM4wIDAQAB"
	DefaultTimeout          = time.Second * 30
)
