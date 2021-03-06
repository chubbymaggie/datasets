package wmh

import (
	"math"

	"fmt"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

const maxUint16 = 65536

// WeightedMinHasher calculates Weighted MinHash-es.
// https://ekzhu.github.io/datasketch/weightedminhash.html
type WeightedMinHasher struct {
	// Size of each hash element in bits. Supported values are 16, 32 and 64.
	Bitness int

	dim        int
	sampleSize int
	rs         [][]float32
	lnCs       [][]float32
	betas      [][]uint16 // attempt to save some memory - [0, 1] is scaled to maxUint16
}

// NewWeightedMinHasher initializes a new instance of WeightedMinHasher.
// `dim` is the bag size.
// `sampleSize` is the hash length.
// `seed` is the random generator seed, as Weighted MinHash is probabilistic.
func NewWeightedMinHasher(dim int, sampleSize int, seed int64) *WeightedMinHasher {
	randSrc := rand.New(rand.NewSource(uint64(seed)))
	gammaGen := distuv.Gamma{Alpha: 2, Beta: 1, Src: randSrc}
	hasher := &WeightedMinHasher{Bitness: 64, dim: dim, sampleSize: sampleSize}
	hasher.rs = make([][]float32, sampleSize)
	for y := 0; y < sampleSize; y++ {
		arr := make([]float32, dim)
		hasher.rs[y] = arr
		for x := 0; x < dim; x++ {
			arr[x] = float32(gammaGen.Rand())
		}
	}
	hasher.lnCs = make([][]float32, sampleSize)
	for y := 0; y < sampleSize; y++ {
		arr := make([]float32, dim)
		hasher.lnCs[y] = arr
		for x := 0; x < dim; x++ {
			arr[x] = float32(math.Log(gammaGen.Rand()))
		}
	}
	uniformGen := distuv.Uniform{Min: 0, Max: 1, Src: randSrc}
	hasher.betas = make([][]uint16, sampleSize)
	for y := 0; y < sampleSize; y++ {
		arr := make([]uint16, dim)
		hasher.betas[y] = arr
		for x := 0; x < dim; x++ {
			arr[x] = uint16(uniformGen.Rand() * maxUint16)
		}
	}
	return hasher
}

// Hash calculates the Weighted MinHash from the weighted bag of features.
// Each feature has an index and a value.
func (wmh *WeightedMinHasher) Hash(values []float32, indices []int) []uint64 {
	hashvalues := make([]uint64, wmh.sampleSize)
	for s := 0; s < wmh.sampleSize; s++ {
		minLnA := math.MaxFloat64
		var k int
		var minT float64
		for vi, j := range indices {
			if j >= wmh.dim {
				panic("index is out of range")
			}
			vlog := math.Log(float64(values[vi]))
			// t = np.floor((vlog / self.rs[i]) + self.betas[i])
			t := math.Floor(vlog/float64(wmh.rs[s][j])) + float64(wmh.betas[s][j])
			// ln_y = (t - self.betas[i]) * self.rs[i]
			lnY := (t - float64(wmh.betas[s][j])) / maxUint16 * float64(wmh.rs[s][j])
			// ln_a = self.ln_cs[i] - ln_y - self.rs[i]
			lnA := float64(wmh.lnCs[s][j]) - lnY - float64(wmh.rs[s][j])
			// k = np.nanargmin(ln_a)
			if lnA < minLnA {
				minLnA = lnA
				k = j
				minT = t
			}
		}
		// hashvalues[i][0], hashvalues[i][1] = k, int(t[k])
		switch wmh.Bitness {
		case 64:
			hashvalues[s] = uint64(uint64(k) | (uint64(minT) << 32))
		case 32:
			hashvalues[s] = uint64(uint32(k) | (uint32(minT) << 16))
		case 16:
			hashvalues[s] = uint64(uint16(k) | (uint16(minT) << 8))
		default:
			panic(fmt.Sprintf("unsupported bitness value: %d", wmh.Bitness))
		}
	}
	return hashvalues
}
