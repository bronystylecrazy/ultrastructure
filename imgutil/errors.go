package imgutil

import (
	"errors"

	"github.com/bronystylecrazy/ultrastructure/imgutil/internal/thumbhash"
)

var ErrInvalidInput = errors.New("imgutil: invalid input")
var ErrOpenFile = errors.New("imgutil: open file")
var ErrDecodeImage = errors.New("imgutil: decode image")
var ErrDecodeHash = errors.New("imgutil: decode hash")
var ErrInvalidHash = thumbhash.ErrInvalidHash
