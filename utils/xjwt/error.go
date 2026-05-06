package xjwt

import (
	"github.com/liangma499/xbase/xerrors"
)

var (
	// indicates JWT token is missing
	ErrMissingToken = xerrors.New("token is missing")

	// indicates JWT token has expired. Can't refresh.
	ErrExpiredToken = xerrors.New("token is expired")

	// indicates auth header is invalid, could for example have the wrong issuer
	ErrInvalidToken = xerrors.New("token is invalid")

	// indicates that there is no corresponding identity information in the payload
	ErrMissingIdentity = xerrors.New("identity is missing")

	// indicates that the same identity is logged in elsewhere
	ErrAuthElsewhere = xerrors.New("auth elsewhere")

	// indicates that the signing method of the token is inconsistent with the configured signing method
	ErrSignAlgorithmNotMatch = xerrors.New("sign algorithm does not match")

	// indicates that the sign algorithm is invalid, must be one of HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384 and ES512
	ErrInvalidSignAlgorithm = xerrors.New("invalid sign algorithm")

	// indicates that the given secret cacheKey is invalid
	ErrInvalidSecretKey = xerrors.New("invalid secret cacheKey")

	// indicates that the given private cacheKey is invalid
	ErrInvalidPrivateKey = xerrors.New("invalid private cacheKey")

	// indicates the given public cacheKey is invalid
	ErrInvalidPublicKey = xerrors.New("invalid public cacheKey")
)

func IsMissingToken(err error) bool {
	return xerrors.Is(err, ErrMissingToken)
}

func IsInvalidToken(err error) bool {
	return xerrors.Is(err, ErrInvalidToken)
}

func IsExpiredToken(err error) bool {
	return xerrors.Is(err, ErrExpiredToken)
}

func IsAuthElsewhere(err error) bool {
	return xerrors.Is(err, ErrAuthElsewhere)
}

func IsIdentityMissing(err error) bool {
	return xerrors.Is(err, ErrMissingIdentity)
}

func IsInvalidSignAlgorithm(err error) bool {
	return xerrors.Is(err, ErrInvalidSignAlgorithm)
}
