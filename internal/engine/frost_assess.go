package engine

// WinterFrostMargin 冬季裕度系数：防冻关注期内调整风险评分。
const WinterFrostMargin = 0.85

// AssessFrost 让季节策略实现 FrostAssessor：
// 防冻季生效时叠加冬季裕度系数，非防冻季按原始观测评分。
func (p FrostPolicy) AssessFrost(in FrostInput) FrostVerdict {
	margin := 1.0
	if p.Active {
		margin = WinterFrostMargin
	}
	verdict := EvaluateFrost(in, margin)
	if p.Active {
		verdict.Detail += " (winter season)"
	}
	return verdict
}
