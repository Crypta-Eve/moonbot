package main

import (
	"strconv"
	"time"
)

const (
	timeFormat = "2006-01-02 15:04:05"
)

type Extraction struct {
	ID                  int    `json:"id"`
	CorporationID       int    `json:"corporation_id"`
	StructureID         int64  `json:"structure_id"`
	MoonID              int    `json:"moon_id"`
	ExtractionStartTime string `json:"extraction_start_time"`
	ChunkArrivalTime    string `json:"chunk_arrival_time"`
	NaturalDecayTime    string `json:"natural_decay_time"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
	Observer            struct {
		CorporationID int    `json:"corporation_id"`
		ObserverID    int64  `json:"observer_id"`
		LastUpdated   string `json:"last_updated"`
		ObserverType  string `json:"observer_type"`
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
		Entries       []struct {
			ID                    int    `json:"id"`
			CorporationID         int    `json:"corporation_id"`
			ObserverID            int64  `json:"observer_id"`
			RecordedCorporationID int    `json:"recorded_corporation_id"`
			CharacterID           int    `json:"character_id"`
			TypeID                int    `json:"type_id"`
			LastUpdated           string `json:"last_updated"`
			Quantity              int    `json:"quantity"`
			CreatedAt             string `json:"created_at"`
			UpdatedAt             string `json:"updated_at"`
		} `json:"entries"`
	} `json:"observer"`
	Moon struct {
		MoonID          int    `json:"moon_id"`
		PlanetID        int    `json:"planet_id"`
		SystemID        int    `json:"system_id"`
		ConstellationID int    `json:"constellation_id"`
		RegionID        int    `json:"region_id"`
		Name            string `json:"name"`
		TypeID          int    `json:"type_id"`
		X               int64  `json:"x"`
		Y               int64  `json:"y"`
		Z               int64  `json:"z"`
		Radius          int    `json:"radius"`
		CelestialIndex  int    `json:"celestial_index"`
		OrbitIndex      int    `json:"orbit_index"`
		SolarSystem     struct {
			SystemID        int     `json:"system_id"`
			ConstellationID int     `json:"constellation_id"`
			RegionID        int     `json:"region_id"`
			Name            string  `json:"name"`
			Security        float64 `json:"security"`
		} `json:"solar_system"`
		Constellation struct {
			ConstellationID int    `json:"constellation_id"`
			RegionID        int    `json:"region_id"`
			Name            string `json:"name"`
		} `json:"constellation"`
		Region struct {
			RegionID int    `json:"region_id"`
			Name     string `json:"name"`
		} `json:"region"`
		MoonReport struct {
			MoonID    int    `json:"moon_id"`
			UserID    int    `json:"user_id"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			Content   []struct {
				TypeID        int         `json:"typeID"`
				GroupID       int         `json:"groupID"`
				TypeName      string      `json:"typeName"`
				Description   string      `json:"description"`
				Mass          int         `json:"mass"`
				Volume        int         `json:"volume"`
				Capacity      int         `json:"capacity"`
				PortionSize   int         `json:"portionSize"`
				RaceID        interface{} `json:"raceID"`
				BasePrice     string      `json:"basePrice"`
				Published     bool        `json:"published"`
				MarketGroupID int         `json:"marketGroupID"`
				IconID        int         `json:"iconID"`
				SoundID       interface{} `json:"soundID"`
				GraphicID     int         `json:"graphicID"`
				Pivot         struct {
					MoonID int    `json:"moon_id"`
					TypeID int    `json:"type_id"`
					Rate   string `json:"rate"`
				} `json:"pivot"`
			} `json:"content"`
		} `json:"moon_report"`
	} `json:"moon"`
	Structure struct {
		CorporationID        int         `json:"corporation_id"`
		StructureID          int64       `json:"structure_id"`
		TypeID               int         `json:"type_id"`
		SystemID             int         `json:"system_id"`
		ProfileID            int         `json:"profile_id"`
		FuelExpires          string      `json:"fuel_expires"`
		StateTimerStart      interface{} `json:"state_timer_start"`
		StateTimerEnd        interface{} `json:"state_timer_end"`
		UnanchorsAt          interface{} `json:"unanchors_at"`
		State                string      `json:"state"`
		ReinforceWeekday     interface{} `json:"reinforce_weekday"`
		ReinforceHour        int         `json:"reinforce_hour"`
		NextReinforceWeekday interface{} `json:"next_reinforce_weekday"`
		NextReinforceHour    int         `json:"next_reinforce_hour"`
		NextReinforceApply   string      `json:"next_reinforce_apply"`
		CreatedAt            string      `json:"created_at"`
		UpdatedAt            string      `json:"updated_at"`
		Info                 struct {
			StructureID   int64   `json:"structure_id"`
			Name          string  `json:"name"`
			OwnerID       int     `json:"owner_id"`
			SolarSystemID int     `json:"solar_system_id"`
			TypeID        int     `json:"type_id"`
			X             float64 `json:"x"`
			Y             float64 `json:"y"`
			Z             float64 `json:"z"`
			CreatedAt     string  `json:"created_at"`
			UpdatedAt     string  `json:"updated_at"`
		} `json:"info"`
	} `json:"structure"`
}

//2021-08-19 01:28:18

func (e Extraction) ChunkArrivalTimeParsed() time.Time {
	// TODO store this for subsequent calls
	t, _ := time.Parse(timeFormat, e.ChunkArrivalTime)
	return t
}

func (e Extraction) StartTimeParsed() time.Time {
	// TODO store this for subsequent calls
	t, _ := time.Parse(timeFormat, e.ExtractionStartTime)
	return t
}

func (e Extraction) DecayTimeParsed() time.Time {
	// TODO store this for subsequent calls
	t, _ := time.Parse(timeFormat, e.NaturalDecayTime)
	return t
}

func (e Extraction) ExtractionTime() time.Duration {
	return e.ChunkArrivalTimeParsed().Sub(e.StartTimeParsed())
}

func (e Extraction) volume() int {
	theoretical := (e.ExtractionTime().Seconds() / 3600.00) * 40000

	per := 0.0
	for _, p := range e.Moon.MoonReport.Content {
		pe, err := strconv.ParseFloat(p.Pivot.Rate, 32)
		if err != nil {
			continue
		}
		per += pe
	}

	return int(theoretical * per)
}

func (e Extraction) isActive() bool {

	return time.Now().After(e.ChunkArrivalTimeParsed()) &&
		time.Now().Before(e.ChunkArrivalTimeParsed().Add(172800*time.Second))
}

type MoonBotResponse []Extraction

func (r MoonBotResponse) Len() int {
	return len(r)
}

func (r MoonBotResponse) Less(i, j int) bool {
	return r[i].ChunkArrivalTimeParsed().Before(r[j].ChunkArrivalTimeParsed())
}

func (r MoonBotResponse) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
