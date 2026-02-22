package imgutil

import (
	"context"
	"image"
	"io"
	"mime/multipart"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
)

type ThumbHasher interface {
	EncodeThumbHash(ctx context.Context, img image.Image) []byte
	EncodeThumbHashBase64(ctx context.Context, img image.Image) string
	EncodeThumbHashResult(ctx context.Context, img image.Image) (ThumbHashResult, error)
	EncodeThumbHashFromBytes(ctx context.Context, data []byte) ([]byte, error)
	EncodeThumbHashBase64FromBytes(ctx context.Context, data []byte) (string, error)
	EncodeThumbHashResultFromBytes(ctx context.Context, data []byte) (ThumbHashResult, error)
	EncodeThumbHashFromReader(ctx context.Context, r io.Reader) ([]byte, error)
	EncodeThumbHashBase64FromReader(ctx context.Context, r io.Reader) (string, error)
	EncodeThumbHashResultFromReader(ctx context.Context, r io.Reader) (ThumbHashResult, error)
	DecodeThumbHash(ctx context.Context, hashData []byte) (image.Image, error)
	DecodeThumbHashFromBytes(ctx context.Context, data []byte) (image.Image, error)
	DecodeThumbHashFromBytesWithCfg(ctx context.Context, data []byte, cfg DecodingCfg) (image.Image, error)
	DecodeThumbHashFromReader(ctx context.Context, r io.Reader) (image.Image, error)
	DecodeThumbHashFromReaderWithCfg(ctx context.Context, r io.Reader, cfg DecodingCfg) (image.Image, error)
	DecodeThumbHashWithCfg(ctx context.Context, hashData []byte, cfg DecodingCfg) (image.Image, error)
	DecodeThumbHashBase64(ctx context.Context, hashBase64 string) (image.Image, error)
	DecodeThumbHashBase64WithCfg(ctx context.Context, hashBase64 string, cfg DecodingCfg) (image.Image, error)
	DecodeThumbHashResult(ctx context.Context, hashData []byte) (DecodedThumbHashResult, error)
	DecodeThumbHashResultWithCfg(ctx context.Context, hashData []byte, cfg DecodingCfg) (DecodedThumbHashResult, error)
	DecodeHash(ctx context.Context, hashData []byte, cfg *DecodingCfg) (*Hash, error)
	DecodeHashBase64(ctx context.Context, hashBase64 string, cfg *DecodingCfg) (*Hash, error)
	EncodeHash(ctx context.Context, hash *Hash) ([]byte, error)
	EncodeHashBase64(ctx context.Context, hash *Hash) (string, error)
	HashSize(ctx context.Context, hash *Hash, baseSize int) (int, int, error)
	GenerateThumbHash(ctx context.Context, fileHeader *multipart.FileHeader) (ThumbHashResult, error)
}

type Service struct {
	otel.Telemetry
}

func NewService() *Service {
	return &Service{
		Telemetry: otel.Nop(),
	}
}

func (s *Service) EncodeThumbHash(ctx context.Context, img image.Image) []byte {
	_, span := s.Obs.Start(ctx, "thumbhash.encode")
	defer span.End()

	return EncodeThumbHash(ctx, img)
}

func (s *Service) EncodeThumbHashBase64(ctx context.Context, img image.Image) string {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_base64")
	defer span.End()

	return EncodeThumbHashBase64(ctx, img)
}

func (s *Service) EncodeThumbHashResult(ctx context.Context, img image.Image) (ThumbHashResult, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_result")
	defer span.End()

	return EncodeThumbHashResult(ctx, img)
}

func (s *Service) EncodeThumbHashFromBytes(ctx context.Context, data []byte) ([]byte, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_from_bytes")
	defer span.End()

	return EncodeThumbHashFromBytes(ctx, data)
}

func (s *Service) EncodeThumbHashBase64FromBytes(ctx context.Context, data []byte) (string, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_base64_from_bytes")
	defer span.End()

	return EncodeThumbHashBase64FromBytes(ctx, data)
}

func (s *Service) EncodeThumbHashResultFromBytes(ctx context.Context, data []byte) (ThumbHashResult, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_result_from_bytes")
	defer span.End()

	return EncodeThumbHashResultFromBytes(ctx, data)
}

func (s *Service) EncodeThumbHashFromReader(ctx context.Context, r io.Reader) ([]byte, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_from_reader")
	defer span.End()

	return EncodeThumbHashFromReader(ctx, r)
}

func (s *Service) EncodeThumbHashBase64FromReader(ctx context.Context, r io.Reader) (string, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_base64_from_reader")
	defer span.End()

	return EncodeThumbHashBase64FromReader(ctx, r)
}

func (s *Service) EncodeThumbHashResultFromReader(ctx context.Context, r io.Reader) (ThumbHashResult, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_result_from_reader")
	defer span.End()

	return EncodeThumbHashResultFromReader(ctx, r)
}

func (s *Service) DecodeThumbHash(ctx context.Context, hashData []byte) (image.Image, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode")
	defer span.End()

	return DecodeThumbHash(ctx, hashData)
}

func (s *Service) DecodeThumbHashFromBytes(ctx context.Context, data []byte) (image.Image, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_from_bytes")
	defer span.End()

	return DecodeThumbHashFromBytes(ctx, data)
}

func (s *Service) DecodeThumbHashFromBytesWithCfg(ctx context.Context, data []byte, cfg DecodingCfg) (image.Image, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_from_bytes_with_cfg")
	defer span.End()

	return DecodeThumbHashFromBytesWithCfg(ctx, data, cfg)
}

func (s *Service) DecodeThumbHashFromReader(ctx context.Context, r io.Reader) (image.Image, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_from_reader")
	defer span.End()

	return DecodeThumbHashFromReader(ctx, r)
}

func (s *Service) DecodeThumbHashFromReaderWithCfg(ctx context.Context, r io.Reader, cfg DecodingCfg) (image.Image, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_from_reader_with_cfg")
	defer span.End()

	return DecodeThumbHashFromReaderWithCfg(ctx, r, cfg)
}

func (s *Service) DecodeThumbHashWithCfg(ctx context.Context, hashData []byte, cfg DecodingCfg) (image.Image, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_with_cfg")
	defer span.End()

	return DecodeThumbHashWithCfg(ctx, hashData, cfg)
}

func (s *Service) DecodeThumbHashBase64(ctx context.Context, hashBase64 string) (image.Image, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_base64")
	defer span.End()

	return DecodeThumbHashBase64(ctx, hashBase64)
}

func (s *Service) DecodeThumbHashBase64WithCfg(ctx context.Context, hashBase64 string, cfg DecodingCfg) (image.Image, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_base64_with_cfg")
	defer span.End()

	return DecodeThumbHashBase64WithCfg(ctx, hashBase64, cfg)
}

func (s *Service) DecodeThumbHashResult(ctx context.Context, hashData []byte) (DecodedThumbHashResult, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_result")
	defer span.End()

	return DecodeThumbHashResult(ctx, hashData)
}

func (s *Service) DecodeThumbHashResultWithCfg(ctx context.Context, hashData []byte, cfg DecodingCfg) (DecodedThumbHashResult, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_result_with_cfg")
	defer span.End()

	return DecodeThumbHashResultWithCfg(ctx, hashData, cfg)
}

func (s *Service) DecodeHash(ctx context.Context, hashData []byte, cfg *DecodingCfg) (*Hash, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_hash")
	defer span.End()

	return DecodeHash(ctx, hashData, cfg)
}

func (s *Service) DecodeHashBase64(ctx context.Context, hashBase64 string, cfg *DecodingCfg) (*Hash, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.decode_hash_base64")
	defer span.End()

	return DecodeHashBase64(ctx, hashBase64, cfg)
}

func (s *Service) EncodeHash(ctx context.Context, hash *Hash) ([]byte, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_hash")
	defer span.End()

	return EncodeHash(ctx, hash)
}

func (s *Service) EncodeHashBase64(ctx context.Context, hash *Hash) (string, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.encode_hash_base64")
	defer span.End()

	return EncodeHashBase64(ctx, hash)
}

func (s *Service) HashSize(ctx context.Context, hash *Hash, baseSize int) (int, int, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.hash_size")
	defer span.End()

	return HashSize(ctx, hash, baseSize)
}

func (s *Service) GenerateThumbHash(ctx context.Context, fileHeader *multipart.FileHeader) (ThumbHashResult, error) {
	_, span := s.Obs.Start(ctx, "thumbhash.generate")
	defer span.End()

	return GenerateThumbHash(ctx, fileHeader)
}

func Providers(extends ...di.Node) di.Node {
	return di.Options(
		di.Provide(NewService, di.As[ThumbHasher](), di.Self(), otel.Layer("thumbhash")),
		di.Options(di.ConvertAnys(extends)...),
	)
}
