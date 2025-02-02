package riskmatrix

// RiskEntry represents a single risk item in the risk matrix
// Includes details about likelihood, impact, and the calculated risk level

type RiskEntry struct {
	ID         string
	Likelihood float32
	Impact     float32
	Category   string	//core and impact