package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
)

const (
	cacheFile  = "cache/seat_data_cache.json"
	reportFile = "seat_report.txt"
)

// --- Structs for parsing ---

type SeatPOI struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type SeatMap struct {
	POIs []SeatPOI `json:"POIs"`
}

type RoomNode struct {
	RoomName string   `json:"roomName"`
	SeatMap  *SeatMap `json:"seatMap"`
}

type AllContentChild struct {
	Children struct {
		Children []any `json:"children"`
	} `json:"children"`
}

type FullData struct {
	AllContent struct {
		Children []any `json:"children"`
	} `json:"allContent"`
}

// --- End of Structs ---

type RoomInfo struct {
	Name  string
	Seats []SeatPOI
}

func main() {
	log.Println("Starting seat ID-Title report generator...")

	file, err := os.Open(cacheFile)
	if err != nil {
		log.Fatalf("Failed to open cache file '%s': %v", cacheFile, err)
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read cache file: %v", err)
	}

	var fullData FullData
	if err := json.Unmarshal(body, &fullData); err != nil {
		log.Fatalf("Failed to parse cache file JSON: %v", err)
	}
	log.Println("Cache file parsed successfully.")

	log.Println("Extracting all seat information...")
	allRooms := extractAllRooms(&fullData)
	if len(allRooms) == 0 {
		log.Fatal("Extraction failed: No rooms with seats found.")
	}

	log.Printf("Generating report file: %s...", reportFile)
	err = generateSeatReport(allRooms)
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	log.Printf("Report generated successfully! Please check %s.", reportFile)
}

func extractAllRooms(data *FullData) []*RoomInfo {
	var roomInfos []*RoomInfo

	// Navigate down the nested structure as described by the user
	for _, l1Child := range data.AllContent.Children {
		l1ChildMap, ok := l1Child.(map[string]any)
		if !ok {
			continue
		}

		if l2Children, ok := l1ChildMap["children"].(map[string]any); ok {
			if l3Children, ok := l2Children["children"].([]any); ok {
				// This is the deepest level where the room items are
				for _, roomItem := range l3Children {
					roomMap, ok := roomItem.(map[string]any)
					if !ok {
						continue
					}

					if uiType, _ := roomMap["ui_type"].(string); uiType == "ht.Seat.RecommendSeatItem" {
						bytes, err := json.Marshal(roomMap)
						if err != nil {
							continue
						}

						var room RoomNode
						if json.Unmarshal(bytes, &room) == nil && room.SeatMap != nil && len(room.SeatMap.POIs) > 0 {
							roomInfos = append(roomInfos, &RoomInfo{
								Name:  room.RoomName,
								Seats: room.SeatMap.POIs,
							})
						}
					}
				}
			}
		}
	}
	return roomInfos
}

func generateSeatReport(rooms []*RoomInfo) error {
	file, err := os.Create(reportFile)
	if err != nil {
		return err
	}
	defer file.Close()

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Name < rooms[j].Name
	})

	_, _ = file.WriteString("# Seat ID to Title Mapping Report\n\n")

	for _, room := range rooms {
		if len(room.Seats) == 0 {
			continue
		}
		_, _ = file.WriteString(fmt.Sprintf("# Room: %s\n", room.Name))

		sort.Slice(room.Seats, func(i, j int) bool {
			titleI, _ := strconv.Atoi(room.Seats[i].Title)
			titleJ, _ := strconv.Atoi(room.Seats[j].Title)
			return titleI < titleJ
		})

		for _, seat := range room.Seats {
			line := fmt.Sprintf("SeatID: %s, Title: %s\n", seat.ID, seat.Title)
			if _, err := file.WriteString(line); err != nil {
				return err
			}
		}
		_, _ = file.WriteString("\n")
	}
	return nil
}
