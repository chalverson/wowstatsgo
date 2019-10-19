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

	var wanted = new(Stat)
	wanted.AchievementPoints = 21835
	wanted.ExaltedReps = 95
	wanted.PetsCollected = 1227
	wanted.MountsCollected = 260
	wanted.QuestsCompleted = 21216
	wanted.FishCaught = 22306
	wanted.PetBattlesWon = 1318
	wanted.PetBattlesPvpWon = 34
	wanted.ItemLevel = 415
	wanted.HonorableKills = 11425
	wanted.LastModified = 1571309922000

	if stats.AchievementPoints != wanted.AchievementPoints {
		t.Errorf("Achievement points incorrect, want %v got %v", wanted.AchievementPoints, stats.AchievementPoints)
	}

	if stats.ExaltedReps != wanted.ExaltedReps {
		t.Errorf("ExaltedReps incorrect, want %v got %v", wanted.ExaltedReps, stats.ExaltedReps)
	}

	if stats.PetsCollected != wanted.PetsCollected {
		t.Errorf("PetsCollected is incorrect, want %v got %v", wanted.PetsCollected, stats.PetsCollected)
	}

	if stats.MountsCollected != wanted.MountsCollected {
		t.Errorf("MountsCollected is incorrect, want %v got %v", wanted.MountsCollected, stats.MountsCollected)
	}

	if stats.QuestsCompleted != wanted.QuestsCompleted {
		t.Errorf("QuestsCompleted is incorrect, want %v got %v", wanted.QuestsCompleted, stats.QuestsCompleted)
	}

	if stats.FishCaught != wanted.FishCaught {
		t.Errorf("FishCaught is incorrect, want %v got %v", wanted.FishCaught, stats.FishCaught)
	}

	if stats.PetBattlesWon != wanted.PetBattlesWon {
		t.Errorf("PetBattlesWon is incorrect, want %v got %v", wanted.PetBattlesWon, stats.PetBattlesWon)
	}

	if stats.PetBattlesPvpWon != wanted.PetBattlesPvpWon {
		t.Errorf("PetBattlesPvpWon is incorrect, want %v got %v", wanted.PetBattlesPvpWon, stats.PetBattlesPvpWon)
	}

	if stats.ItemLevel != wanted.ItemLevel {
		t.Errorf("ItemLevel is incorrect, want %v got %v", wanted.ItemLevel, stats.ItemLevel)
	}

	if stats.HonorableKills != wanted.HonorableKills {
		t.Errorf("HonorableKills is incorrect, want %v got %v", wanted.HonorableKills, stats.HonorableKills)
	}

	if stats.LastModified != wanted.LastModified {
		t.Errorf("LastModified is incorrect, want %v got %v", wanted.LastModified, stats.LastModified)
	}
}
