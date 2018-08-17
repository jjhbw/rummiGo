package rummikub

type Rules struct {
	JokersPerCombination int      `json:"jokers_per_combination"`
	Values               int      `json:"values"`
	JokersInPlay         int      `json:"jokers_in_play"`
	Colors               []string `json:"colors"`
	Replicates           int      `json:"replicates"`
	StartingHandSize     int      `json:"starting_hand_size"`

	// minimum summed value of a first move
	FirstMoveValue int `json:"first_move_value"`
}

// NewDefaultRules returns the default game rules.
// Primarily useful to reduce the verbosity of unit tests.
func NewDefaultRules() Rules {
	return Rules{
		Colors:               []string{"red", "green", "blue", "yellow"},
		Values:               13,
		JokersPerCombination: 1,
		JokersInPlay:         2,
		StartingHandSize:     14,
		Replicates:           2,
		FirstMoveValue:       14,
	}
}

// BaseBricks gets all the UNIQUE getBricks that are in the Rummikub play set described by the receiving Rules struct.
// NOTE: EXCEPT the jokers! (for flexibility)
func (g Rules) BaseBricks() []Brick {
	bricks := []Brick{}

	// '_' the blank identifier references the index of each iteration
	for val := 1; val <= g.Values; val++ {
		for _, col := range g.Colors {
			bricks = append(bricks, Brick{Value: val, Color: col})
		}
	}
	return bricks
}

// AllBricks returns all bricks that are in the Rummikub play set described by the receiving Rules struct.
// Note that all color-value combinations are returned in the appropriate quantities (defined by Rules.replicates).
// INCLUDING the jokers.
func (g Rules) AllBricks() []Brick {
	bricks := []Brick{}

	// Add the appropriate number of replicate base bricks.
	for i := 0; i < g.Replicates; i++ {
		bricks = append(bricks, g.BaseBricks()...)
	}

	// Add the appropriate number of jokers.
	for j := 0; j < g.JokersInPlay; j++ {
		bricks = append(bricks, MakeJoker())
	}

	return bricks
}

func (g *Rules) IsLegalCombination(c BrickCombination) (bool, string) {
	// test if a combination is legal given the game rules
	// resistant to user input

	// test if brick colors are legal according to the game rules
	for _, brick := range c.getBricks() {
		ok := false
		for _, color := range g.Colors {
			if brick.Color == color || brick.Color == JokerColor {
				ok = true
				break
			}
		}
		if !ok {
			return false, UNKNOWN_COLOR
		}
	}

	// test if brick values are legal according to the game rules
	for _, brick := range c.getBricks() {
		if brick.Value > g.Values || brick.Value < 1 {
			return false, VALUE_OUT_OF_BOUNDS
		}
	}

	// test amount of jokers according to the game rules
	jokercount := 0
	for _, brick := range c.getBricks() {
		if brick.Color == JokerColor {
			jokercount++
		}
		if jokercount > g.JokersPerCombination {
			return false, TOO_MANY_JOKERS_IN_COMBINATION
		}
	}

	// Test combination-level validity. We ignore the reasons why the combination may not be valid.
	isRun, _ := c.IsValidRun()
	isGroup, _ := c.IsValidGroup()

	//NOTE: some combinations qualify as groups AND as sets, such as [(joker)(joker)(1, red)], depending on the JokersPerCombination setting.
	if !(isRun || isGroup) {
		return false, ILLEGAL_COMBINATION
	}

	return true, LEGAL_COMBINATION
}
