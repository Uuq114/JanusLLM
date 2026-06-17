package auth

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
)

type Team struct {
	TeamId         int         `gorm:"primaryKey;column:team_id"`
	TeamName       string      `gorm:"column:team_name"`
	ModelList      StringSlice `gorm:"column:model_list"`
	OrganizationId int         `gorm:"column:organization_id"`
}

func CreateTeamRecord(teamName string, organizationName string) {
	if err := CreateTeamRecordWithError(teamName, organizationName); err != nil {
		log.Printf("CreateTeamRecord: %v", err)
	}
}

func CreateTeamRecordWithError(teamName string, organizationName string) error {
	db, err := connectAuthDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer closeAuthDatabaseConnection(db)
	organization, err := getOrganizationRecord(db, organizationName)
	if err != nil {
		return err
	}
	if organization == nil {
		return fmt.Errorf("organization record not found: %s", organizationName)
	}
	result := db.Table("janus_auth_team").Create(&Team{
		TeamName:       teamName,
		ModelList:      StringSlice{"*"},
		OrganizationId: organization.OrganizationId,
	})
	if result.Error != nil {
		return fmt.Errorf("create team record: %w", result.Error)
	}
	return nil
}

func GetTeamRecord(teamName string) *Team {
	team, err := GetTeamRecordWithError(teamName)
	if err != nil {
		log.Printf("GetTeamRecord: %v", err)
		return nil
	}
	return team
}

func GetTeamRecordWithError(teamName string) (*Team, error) {
	db, err := connectAuthDatabase()
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	defer closeAuthDatabaseConnection(db)
	return getTeamRecord(db, teamName)
}

func getTeamRecord(db *gorm.DB, teamName string) (*Team, error) {
	var team Team
	result := db.Table("janus_auth_team").Where("team_name = ?", teamName).First(&team)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get team record: %w", result.Error)
	}
	return &team, nil
}

// update team's organization, not team name

func UpdateTeamRecord(teamName string, organizationName string) {
	if err := UpdateTeamRecordWithError(teamName, organizationName); err != nil {
		log.Printf("UpdateTeamRecord: %v", err)
	}
}

func UpdateTeamRecordWithError(teamName string, organizationName string) error {
	db, err := connectAuthDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer closeAuthDatabaseConnection(db)
	team, err := getTeamRecord(db, teamName)
	if err != nil {
		return err
	}
	if team == nil {
		return fmt.Errorf("team record not found: %s", teamName)
	}
	organization, err := getOrganizationRecord(db, organizationName)
	if err != nil {
		return err
	}
	if organization == nil {
		return fmt.Errorf("organization record not found: %s", organizationName)
	}
	team.OrganizationId = organization.OrganizationId
	result := db.Table("janus_auth_team").Save(team)
	if result.Error != nil {
		return fmt.Errorf("update team record: %w", result.Error)
	}
	return nil
}

func DeleteTeamRecord(teamName string) {
	if err := DeleteTeamRecordWithError(teamName); err != nil {
		log.Printf("DeleteTeamRecord: %v", err)
	}
}

func DeleteTeamRecordWithError(teamName string) error {
	db, err := connectAuthDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer closeAuthDatabaseConnection(db)
	team, err := getTeamRecord(db, teamName)
	if err != nil {
		return err
	}
	if team == nil {
		return fmt.Errorf("team record not found: %s", teamName)
	}
	result := db.Table("janus_auth_team").Delete(team)
	if result.Error != nil {
		return fmt.Errorf("delete team record: %w", result.Error)
	}
	return nil
}
