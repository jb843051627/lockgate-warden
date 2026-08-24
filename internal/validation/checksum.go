package validation

import (
	"math"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// ComputeChecksum 计算批次校验和：
// 传感器编码、序列号、采样时刻（Unix 秒）与读数依次混入 FNV-1a。
func ComputeChecksum(points []model.TelemetryPointInput) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	mix := func(b byte) {
		h ^= uint32(b)
		h *= prime32
	}
	for _, p := range points {
		for _, c := range []byte(p.SensorCode) {
			mix(c)
		}
		mix(byte(p.Seq))
		mix(byte(p.Seq >> 8))
		mix(byte(p.Seq >> 16))
		mix(byte(p.Seq >> 24))
		sec := p.TakenAt.Unix()
		mix(byte(sec))
		mix(byte(sec >> 8))
		mix(byte(sec >> 24))
		bits := math.Float32bits(float32(p.Value))
		mix(byte(bits))
		mix(byte(bits >> 8))
		mix(byte(bits >> 16))
		mix(byte(bits >> 24))
	}
	return h
}

// VerifyChecksum 比对请求携带的校验和与重算结果。
func VerifyChecksum(points []model.TelemetryPointInput, got uint32) error {
	if ComputeChecksum(points) != got {
		return model.ErrChecksum
	}
	return nil
}
