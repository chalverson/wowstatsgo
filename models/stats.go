package models

import "time"

// Map the stats table. This holds the "interesting" stats that I'm interested in.
type Stats struct {
	Toon             *Toon
	LastModified     int64
	CreateDate       time.Time
	Level            int64
	AchievementPoint int64
	ExaltedReps      int64
	MountsCollected  int64
	QuestsCompleted  int64
	FishCaught       int64
	PetsCollected    int64
	PetBattlesWon    int64
	PetBattlesPvpWon int64
	ItemLevel        int64
	HonorableKills   int64
}

// Insert a stats record. This doesn't check to see if a duplicate exists, it relies on the database's
// constraints to handle that.
func (db *WowDB) InsertStats(stats *Stats) error {
	_, err := db.Exec("INSERT INTO stats (toon_id, last_modified, create_date, level, achievement_points, number_exalted, mounts_owned, quests_completed, fish_caught, pets_owned, pet_battles_won, pet_battles_pvp_won, item_level, honorable_kills) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)",
		stats.Toon.Id, stats.LastModified, stats.CreateDate, stats.Level, stats.AchievementPoint, stats.ExaltedReps, stats.MountsCollected, stats.QuestsCompleted,
		stats.FishCaught, stats.PetsCollected, stats.PetBattlesWon, stats.PetBattlesPvpWon, stats.ItemLevel, stats.HonorableKills)

	if err != nil {
		return err
	}
	return nil
}

// Get a list of the latest Stats for all toons. This will get just the latest day's stats which is useful
// for email or CLI.
func (db *WowDB) GetAllToonLatestQuickSummary() []Stats {
	rows, _ := db.Query("select t.id, s.level, s.item_level, s.create_date, t.name, s.last_modified from stats s join toon t on s.toon_id = t.id and s.create_date::date = (select max(create_date::date) from stats) ORDER BY s.level, s.item_level DESC, t.name ASC")
	defer rows.Close()
	var stats []Stats
	for rows.Next() {
		var id int64
		var s Stats
		var name string
		rows.Scan(&id, &s.Level, &s.ItemLevel, &s.CreateDate, &name, &s.LastModified)
		s.Toon, _ = db.GetToonById(id)
		stats = append(stats, s)
	}
	return stats
}

// Get the LastModified field as a human readable format as YYYY-MM-DD HH:MM:SS.
// Need to pull this out separately because we have extra detail from the JSON. We need to divide the time
// by 1000, then format it.
func (s *Stats) LastModifiedAsDateTime() string {
	t := time.Unix(s.LastModified/1000, 0)
	return t.UTC().Format("2006-01-02 15:04:05")
}
