package main

import (
	"io/ioutil"
	"testing"
)

func TestParseStatsFromJson(t *testing.T) {
	jsonText, err := ioutil.ReadFile("test-json.json")
	if err != nil {
		t.Error("Could not read file")
	}

	stats := ParseStatsFromJson(string(jsonText))

	if stats.AchievementPoints != 21835 {
		t.Errorf("Achievement points incorrect, want %v got %v", 21834, stats.AchievementPoints)
	}

	if stats.ExaltedReps != 95 {
		t.Errorf("ExaltedReps incorrect, want %v got %v", 95, stats.ExaltedReps)
	}

	if stats.PetsCollected != 1227 {
		t.Errorf("PetsCollected is incorrect, want %v got %v", 1227, stats.PetsCollected)
	}

	if stats.MountsCollected != 260 {
		t.Errorf("MountsCollected is incorrect, want %v got %v", 260, stats.MountsCollected)
	}

	if stats.QuestsCompleted != 21216 {
		t.Errorf("QuestsCompleted is incorrect, want %v got %v", 21216, stats.QuestsCompleted)
	}

	if stats.FishCaught != 22306 {
		t.Errorf("FishCaught is incorrect, want %v got %v", 22306, stats.FishCaught)
	}

	if stats.PetBattlesWon != 1318 {
		t.Errorf("PetBattlesWon is incorrect, want %v got %v", 1318, stats.PetBattlesWon)
	}

	if stats.PetBattlesPvpWon != 34 {
		t.Errorf("PetBattlesPvpWon is incorrect, want %v got %v", 34, stats.PetBattlesPvpWon)
	}

	if stats.ItemLevel != 415 {
		t.Errorf("ItemLevel is incorrect, want %v got %v", 415, stats.ItemLevel)
	}

	if stats.HonorableKills != 11425 {
		t.Errorf("HonorableKills is incorrect, want %v got %v", 11425, stats.HonorableKills)
	}

	if stats.LastModified != 1571309922000 {
		t.Errorf("LastModified is incorrect, want %v got %v", 1571309922000, stats.LastModified)
	}
}
