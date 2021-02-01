package captcha

import "errors"

var (
	ErrCaptchaFailed   = errors.New("captcha failed")
	ErrCaptchaRequired = errors.New("captcha required")
)
