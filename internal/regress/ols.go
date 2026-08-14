// Package regress implements plain multiple linear regression (OLS) via the
// normal equations, using only the standard library. This avoids depending
// on gonum or similar, since environments without access to the Go module
// proxy can't `go get` third-party packages.
//
// For a design matrix X (n x k, first column typically all-ones for the
// intercept) and response y (n), OLS solves:
//
//	beta = (X^T X)^-1 X^T y
//
// This is fine for the modest matrix sizes here (a few dozen columns from
// neighborhood/category dummies). For much larger or ill-conditioned
// problems you'd want QR decomposition instead, but normal equations are
// simple, dependency-free, and adequate at this scale.
package regress

import "fmt"

// Fit computes OLS coefficients for X (n rows x k cols) and y (n).
func Fit(X [][]float64, y []float64) ([]float64, error) {
	n := len(X)
	if n == 0 {
		return nil, fmt.Errorf("empty design matrix")
	}
	k := len(X[0])
	if len(y) != n {
		return nil, fmt.Errorf("X has %d rows but y has %d", n, len(y))
	}

	// XtX (k x k) and Xty (k)
	xtx := make([][]float64, k)
	for i := range xtx {
		xtx[i] = make([]float64, k)
	}
	xty := make([]float64, k)

	for row := 0; row < n; row++ {
		for i := 0; i < k; i++ {
			xty[i] += X[row][i] * y[row]
			for j := 0; j < k; j++ {
				xtx[i][j] += X[row][i] * X[row][j]
			}
		}
	}

	beta, err := solveLinearSystem(xtx, xty)
	if err != nil {
		return nil, fmt.Errorf("solving normal equations (check for collinear/duplicate columns): %w", err)
	}
	return beta, nil
}

// Predict returns X * beta.
func Predict(X [][]float64, beta []float64) []float64 {
	out := make([]float64, len(X))
	for i, row := range X {
		var s float64
		for j, v := range row {
			s += v * beta[j]
		}
		out[i] = s
	}
	return out
}

// RSquared computes the coefficient of determination.
func RSquared(y, yHat []float64) float64 {
	var mean float64
	for _, v := range y {
		mean += v
	}
	mean /= float64(len(y))

	var ssRes, ssTot float64
	for i := range y {
		ssRes += (y[i] - yHat[i]) * (y[i] - yHat[i])
		ssTot += (y[i] - mean) * (y[i] - mean)
	}
	if ssTot == 0 {
		return 0
	}
	return 1 - ssRes/ssTot
}

// solveLinearSystem solves A x = b via Gauss-Jordan elimination with partial
// pivoting. A is modified in place (on a copy); dimension is small in this
// application (tens of columns) so no need for anything fancier.
func solveLinearSystem(a [][]float64, b []float64) ([]float64, error) {
	n := len(a)
	// augmented matrix
	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, n+1)
		copy(aug[i], a[i])
		aug[i][n] = b[i]
	}

	for col := 0; col < n; col++ {
		// partial pivot
		pivotRow := col
		maxAbs := abs(aug[col][col])
		for r := col + 1; r < n; r++ {
			if v := abs(aug[r][col]); v > maxAbs {
				maxAbs = v
				pivotRow = r
			}
		}
		if maxAbs < 1e-10 {
			return nil, fmt.Errorf("singular matrix at column %d", col)
		}
		aug[col], aug[pivotRow] = aug[pivotRow], aug[col]

		pivot := aug[col][col]
		for c := col; c <= n; c++ {
			aug[col][c] /= pivot
		}
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			factor := aug[r][col]
			if factor == 0 {
				continue
			}
			for c := col; c <= n; c++ {
				aug[r][c] -= factor * aug[col][c]
			}
		}
	}

	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = aug[i][n]
	}
	return x, nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
