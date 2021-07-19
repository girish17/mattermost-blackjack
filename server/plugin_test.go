package main

import (
	"testing"
)

func TestCalculateScore(t *testing.T)  {
	var cardsDealt []string
	if res := calculateScore(cardsDealt); res != 0 {
		t.Log("error should be 0, but got", res)
		t.Fail()
	}

	cardsDealt = []string{"ace_of_hearts", "ace_of_spades", "queen_of_hearts"}
	if res := calculateScore(cardsDealt); res != 12 {
		t.Log("error should be 12, but got", res)
		t.Fail()
	}

	cardsDealt = []string{"ace_of_hearts", "ace_of_spades", "ace_of_clubs", "ace_of_spades"}
	if res := calculateScore(cardsDealt); res != 14 {
		t.Log("error should be 14, but got", res)
		t.Fail()
	}

	cardsDealt = []string{"ace_of_hearts", "ace_of_spades", "ace_of_clubs", "ace_of_spades", "queen_of_hearts"}
	if res := calculateScore(cardsDealt); res != 14 {
		t.Log("error should be 14, but got", res)
		t.Fail()
	}

	cardsDealt = []string{"ace_of_hearts", "ace_of_spades", "queen_of_hearts", "ace_of_clubs", "ace_of_spades"}
	if res := calculateScore(cardsDealt); res != 14 {
		t.Log("error should be 14, but got", res)
		t.Fail()
	}

	cardsDealt = []string{"2_of_clubs", "6_of_clubs", "3_of_clubs", "2_of_hearts", "8_of_diamonds"}
	if res := calculateScore(cardsDealt); res != 21 {
		t.Log("error should be 21, but got", res)
		t.Fail()
	}
}
