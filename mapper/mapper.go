package mapper

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type SeatInfo struct {
	SeatID int
	Title  string
}

type SeatMapper map[string][]SeatInfo

var seatMap SeatMapper

func LoadSeatMap(path string) (SeatMapper, error) {
	//打开座位文件
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open seat map file: %w", err)
	}
	defer file.Close()

	mapper := make(SeatMapper)
	var currentRoom string
	//创建匹配器，用于正则匹配
	roomRegex := regexp.MustCompile(`^# Room: (.+)$`)
	seatRegex := regexp.MustCompile(`^SeatID: (\d+), Title: (.+)$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if roomMatch := roomRegex.FindStringSubmatch(line); roomMatch != nil {
			currentRoom = strings.TrimSpace(roomMatch[1])
			if _, ok := mapper[currentRoom]; !ok {
				mapper[currentRoom] = []SeatInfo{}
			}
		} else if seatMatch := seatRegex.FindStringSubmatch(line); seatMatch != nil && currentRoom != "" {
			seatID, _ := strconv.Atoi(seatMatch[1])
			title := strings.TrimSpace(seatMatch[2])
			mapper[currentRoom] = append(mapper[currentRoom], SeatInfo{
				SeatID: seatID,
				Title:  title,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading seat map file: %w", err)
	}
	seatMap = mapper
	return mapper, nil
}

func GetSeatID(roomName string, seatTitle string) (int, error) {
	if seatMap == nil {
		return 0, fmt.Errorf("seat map is not loaded")
	}

	seats, ok := seatMap[roomName]
	if !ok {
		return 0, fmt.Errorf("room '%s' not found in seat map", roomName)
	}

	for _, seat := range seats {
		if seat.Title == seatTitle {
			return seat.SeatID, nil
		}
	}

	return 0, fmt.Errorf("seat '%s' not found in room '%s'", seatTitle, roomName)
}
