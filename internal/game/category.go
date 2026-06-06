package game

type Category string

const (
	CategoryBullet Category = "bullet"
	CategoryBlitz  Category = "blitz"
	CategoryRapid  Category = "rapid"
)

// bullet < 3min, blitz  3–<10min, rapid >= 10min.
func CategoryForBaseSeconds(base int) Category {
	switch {
	case base < 180:
		return CategoryBullet
	case base < 600:
		return CategoryBlitz
	default:
		return CategoryRapid
	}
}
