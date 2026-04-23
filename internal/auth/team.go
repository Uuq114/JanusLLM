package auth

import (
	"log"

	janusDb "github.com/Uuq114/JanusLLM/internal/db"
)

type Team struct {
	TeamId         int         `gorm:"primaryKey;column:team_id"`
	TeamName       string      `gorm:"column:team_name"`
	ModelList      StringSlice `gorm:"column:model_list"`
	OrganizationId int         `gorm:"column:organization_id"`
}

func CreateTeamRecord(teamName string, organizationName string) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)
	organization := GetOrganizationRecord(organizationName)
	if organization == nil {
		log.Fatal("Organization record not found, name", organizationName)
		return
	}
	result := db.Table("janus_auth_team").Create(&Team{
		TeamName:       teamName,
		ModelList:      StringSlice{"*"},
		OrganizationId: organization.OrganizationId,
	})
	if result.Error != nil {
		log.Fatal("Failed to create team record, err:", result.Error)
		return
	}
}

func GetTeamRecord(teamName string) *Team {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return nil
	}
	defer janusDb.CloseDatabaseConnection(db)
	var team Team
	result := db.Table("janus_auth_team").Where("team_name = ?", teamName).First(&team)
	if result.Error != nil {
		log.Fatal("Failed to get team record, err:", result.Error)
		return nil
	}
	return &team
}

// update team's organization, not team name

func UpdateTeamRecord(teamName string, organizationName string) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)
	var team Team
	result := db.Table("janus_auth_team").Where("team_name = ?", teamName).First(&team)
	if result.Error != nil {
		log.Fatal("Failed to get team record, err:", result.Error)
		return
	}
	organization := GetOrganizationRecord(organizationName)
	if organization == nil {
		log.Fatal("Organization record not found, name", organizationName)
		return
	}
	team.OrganizationId = organization.OrganizationId
	result = db.Table("janus_auth_team").Save(&team)
	if result.Error != nil {
		log.Fatal("Failed to update team record, err:", result.Error)
		return
	}
}

func DeleteTeamRecord(teamName string) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer janusDb.CloseDatabaseConnection(db)
	var team Team
	result := db.Table("janus_auth_team").Where("team_name = ?", teamName).First(&team)
	if result.Error != nil {
		log.Fatal("Failed to get team record, err:", result.Error)
		return
	}
	result = db.Table("janus_auth_team").Delete(&team)
	if result.Error != nil {
		log.Fatal("Failed to delete team record, err:", result.Error)
		return
	}
}
