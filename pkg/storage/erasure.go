package storage

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
)

const (
	DataShards   = 10
	ParityShards = 4
	TotalShards  = DataShards + ParityShards
)

// ErasureCoder يدير تجزئة وإعادة بناء البيانات بأمان
type ErasureCoder struct {
	enc reedsolomon.Encoder
}

// NewErasureCoder ينشئ مشفر تجزيئي جديد
func NewErasureCoder() (*ErasureCoder, error) {
	enc, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, fmt.Errorf("failed to create reed-solomon encoder: %w", err)
	}
	return &ErasureCoder{enc: enc}, nil
}

// Encode يقسم البيانات إلى أجزاء مشفرة
func (e *ErasureCoder) Encode(data []byte) ([][]byte, error) {
	shards := make([][]byte, TotalShards)
	for i := range shards {
		shards[i] = make([]byte, 0)
	}

	// تقسيم البيانات
	shards, err := e.enc.Split(data)
	if err != nil {
		return nil, fmt.Errorf("failed to split data: %w", err)
	}

	// إنشاء أجزاء التكافؤ
	if err := e.enc.Encode(shards); err != nil {
		return nil, fmt.Errorf("failed to encode parity: %w", err)
	}

	return shards, nil
}

// Reconstruct يعيد بناء البيانات الأصلية من الأجزاء المتاحة
func (e *ErasureCoder) Reconstruct(shards [][]byte) ([]byte, error) {
	// إصلاح الأجزاء المفقودة
	if err := e.enc.Reconstruct(shards); err != nil {
		return nil, fmt.Errorf("failed to reconstruct shards: %w", err)
	}

	// التحقق من سلامة البيانات بعد الإصلاح
	if ok, err := e.enc.Verify(shards); err != nil || !ok {
		return nil, fmt.Errorf("data verification failed after reconstruction")
	}

	// حساب الطول الإجمالي للبيانات
	totalLen := 0
	for _, shard := range shards[:DataShards] {
		totalLen += len(shard)
	}

	// دمج الأجزاء للبيانات الأصلية
	buf := make([]byte, totalLen)
	offset := 0
	for _, shard := range shards[:DataShards] {
		copy(buf[offset:], shard)
		offset += len(shard)
	}

	return buf, nil
}
