package rummikub

import (
	"hash/fnv"
	"strconv"
)

const JokerColor string = "joker"

// the central Brick struct.
type Brick struct {
	Value int    `json:"value"` // 1 or higher, jokers are valued at 0.
	Color string `json:"color"` // any valid string.
}

// Hash returns the uint32 hash of the Brick (e.g. for use in de-duplication).
// I consider this method slightly more flexible than simply concatenating the Value and Color fields.
func (b *Brick) Hash() uint32 {
	h := fnv.New32a()

	// add the brick value to the running hash
	// NOTE: from the docs
	// "Write (via the embedded io.Writer interface) adds more data to the running hash.
	// It never returns an error."
	h.Write([]byte(strconv.Itoa(b.Value)))

	// add the brick color to the running hash
	h.Write([]byte(b.Color))

	hashedBrick := h.Sum32()

	return hashedBrick
}

// MakeJoker properly initiates a Joker Brick.
func MakeJoker() Brick {
	return Brick{Value: 1, Color: JokerColor}
}

// returns all getBricks in Brick slice A that are not in Brick slice B.
// (returns B-A)
// Note that it respects any duplicates.
func BrickSliceDiff(a []Brick, b []Brick) []Brick {
	aBrickCounts := make(map[Brick]int)
	bBrickCounts := make(map[Brick]int)
	for _, aStone := range a {
		aBrickCounts[aStone]++
	}
	for _, bStone := range b {
		bBrickCounts[bStone]++
	}

	diff := []Brick{}
	for brick, timesInB := range bBrickCounts {
		timesInA := aBrickCounts[brick]
		delta := timesInB - timesInA
		if delta > 0 {
			for i := 0; i < delta; i++ {
				diff = append(diff, brick)
			}
		}
	}

	return diff
}

// Break up a slice of BrickCombinations into their respective unordered getBricks.
func DissolveCombinations(combinations []BrickCombination) []Brick {
	bricks := []Brick{}
	for _, c := range combinations {
		bricks = append(bricks, c.getBricks()...)
	}
	return bricks
}

type BrickCombination struct {
	// a legal combination of uniqueBricks (a row or a set)
	Bricks []Brick `json:"bricks"`
}

func NewBrickCombination(b ...Brick) BrickCombination {
	new := BrickCombination{}
	new.AddBrick(b...)
	return new
}

func (c *BrickCombination) getBricks() []Brick {
	//simple method to add uniqueBricks to a combination
	return c.Bricks
}

func (c *BrickCombination) AddBrick(b ...Brick) {
	//simple method to add uniqueBricks to a combination
	c.Bricks = append(c.Bricks, b...)
}

func (c *BrickCombination) Copy() BrickCombination {
	// make a copy of the BrickCombination
	return NewBrickCombination(c.Bricks...)
}

type CombinationIdentity uint32

// Hash the BrickCombination (e.game. for use in de-duplication).
//NOTE uses bitwise XOR to ignore brick order.
func (c *BrickCombination) Hash() CombinationIdentity {
	var h uint32
	for _, b := range c.Bricks {
		h = h ^ b.Hash()
	}

	return CombinationIdentity(h)
}

// Contains checks if the combination contains a brick matching the provided value and color
func (c *BrickCombination) Contains(value int, color string) bool {
	for _, a := range c.getBricks() {
		if a.Color == color && a.Value == value {
			return true
		}
	}
	return false
}

// outcomes of matching BrickCombinations to game rules
const (
	COMBINATION_TOO_SMALL     = "combination is smaller than 3"
	VALID_RUN                 = "combination is a valid run"
	VALID_GROUP               = "combination is a valid group"
	CONTAINS_ONLY_JOKERS      = "combination contains only jokers"
	CONTAINS_MULTIPLE_VALUES  = "not a group: combination contains more than one unique value"
	CONTAINS_MULTIPLE_COLORS  = "not a run: combination contains uniqueBricks of more than one color"
	CONTAINS_DUPLICATE_VALUES = "not a run: combination contains duplicate values"
	COLORS_NOT_UNIQUE         = "not a group: combination contains duplicates of a color"
	NOT_CONSECUTIVE           = "not a run: brick values not consecutive or semi-consecutive (i.e. with joker)"
)

func (c *BrickCombination) IsValidGroup() (bool, string) {
	// check if the combination is a valid group

	// is longer than 3?
	// NOTE that runs longer than len(allowedColors) are caught either here or at the game rules legality-level check
	// due to their colors not being unique (this function) or their colors not being in allowedColors (game rules legality check).
	if len(c.Bricks) < 3 {
		return false, COMBINATION_TOO_SMALL
	}

	// contains something else than just jokers?
	// NOTE: testing the number of jokers is for the game-level legality check to decide
	ok := false
	for _, b := range c.Bricks {
		if b.Color != JokerColor {
			ok = true
			break
		}
	}
	if !ok {
		return false, CONTAINS_ONLY_JOKERS
	}

	// are all values the same (ignoring the value of a joker )?
	valueMap := map[int]bool{}
	for _, b := range c.Bricks {
		if b.Color != JokerColor {
			valueMap[b.Value] = true
		}
	}
	if len(valueMap) > 1 { // number of keys
		return false, CONTAINS_MULTIPLE_VALUES
	}

	// are all colors unique?
	// testing whether the colors are in the legal set is up to the game-level legality checker
	colorMap := map[string]bool{}
	for _, b := range c.Bricks {
		if _, ok := colorMap[b.Color]; ok && b.Color != JokerColor {
			return false, COLORS_NOT_UNIQUE
		} else {
			colorMap[b.Color] = true
		}
	}

	return true, VALID_GROUP
}

func (c *BrickCombination) IsValidRun() (bool, string) {
	// check if the combination is a valid run

	// contains at least 3 uniqueBricks?
	if len(c.Bricks) < 3 {
		return false, COMBINATION_TOO_SMALL
	}

	// contains something else than just jokers?
	// NOTE: testing the number of jokers is for the game-level legality check to decide
	ok := false
	for _, b := range c.Bricks {
		if b.Color != JokerColor {
			ok = true
			break
		}
	}
	if !ok {
		return false, CONTAINS_ONLY_JOKERS
	}

	// contains only uniqueBricks of a single color ( not counting JokerColor )
	colorMap := map[string]bool{}
	for _, b := range c.Bricks {
		if b.Color != JokerColor {
			colorMap[b.Color] = true
		}
	}
	if len(colorMap) > 1 { // number of keys
		return false, CONTAINS_MULTIPLE_COLORS
	}

	// does the combination only contain unique values?
	// testing whether the values are in the legal range is up to the game-level legality checker
	valueMap := map[int]bool{}
	for _, b := range c.Bricks {
		if _, ok := valueMap[b.Value]; ok && b.Color != JokerColor {
			return false, CONTAINS_DUPLICATE_VALUES
		} else {
			valueMap[b.Value] = true
		}
	}

	// contains uniqueBricks of consecutive values, possibly interspaced by joker(s)
	// NOTE: does not rely on brick order as presented by c.getBricks()

	// count the amount of jokers in the combination
	jokerCount := 0
	for _, brick := range c.Bricks {
		if brick.Color == JokerColor {
			jokerCount++
		}
	}

	// find the highest and lowest valued uniqueBricks that aren't jokers
	var highest Brick
	highestVal := 0 // initialize at 0
	for _, b := range c.Bricks {
		if b.Value > highestVal && b.Color != JokerColor {
			highest = b
			highestVal = b.Value
		}
	}

	var lowest Brick
	lowestVal := 100000 // initialize at stupidly high
	for _, b := range c.Bricks {
		if b.Value < lowestVal && b.Color != JokerColor {
			lowest = b
			lowestVal = b.Value
		}
	}

	// walk the BrickCombinations count down from the highest value, sacrificing joker counts
	for v := highest.Value; v >= lowest.Value; v-- {
		// if combination does not contain a stone of this value:
		if !c.Contains(v, highest.Color) {
			if jokerCount > 0 {
				jokerCount--
			} else {
				return false, NOT_CONSECUTIVE
			}
		}
	}

	return true, VALID_RUN

}
