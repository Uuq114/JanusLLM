package auth

import "log"

type User struct {
	UserId         int    `gorm:"primaryKey;column:user_id"`
	UserName       string `gorm:"column:user_name"`
	OrganizationId int    `gorm:"column:organization_id"`
}

func CreateUserRecord(userName string, organizationName string) {
	db, err := ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer CloseDatabaseConnection(db)
	organization := GetOrganizationRecord(organizationName)
	if organization == nil {
		log.Fatal("Organization record not found, name", organizationName)
		return
	}
	result := db.Table("janus_auth_user").Create(&User{UserName: userName, OrganizationId: organization.OrganizationId})
	if result.Error != nil {
		log.Fatal("Failed to create user record, err:", result.Error)
		return
	}
}

func GetUserRecord(userName string) *User {
	db, err := ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return nil
	}
	defer CloseDatabaseConnection(db)
	var user User
	result := db.Table("janus_auth_user").Where("user_name = ?", userName).First(&user)
	if result.Error != nil {
		log.Fatal("Failed to get user record, err:", result.Error)
		return nil
	}
	return &user
}

// update user's organization, not username

func UpdateUserRecord(userName string, organizationName string) {
	db, err := ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer CloseDatabaseConnection(db)
	var user User
	result := db.Table("janus_auth_user").Where("user_name = ?", userName).First(&user)
	if result.Error != nil {
		log.Fatal("Failed to get user record, err:", result.Error)
		return
	}
	organization := GetOrganizationRecord(organizationName)
	if organization == nil {
		log.Fatal("Organization record not found, name", organizationName)
		return
	}
	user.OrganizationId = organization.OrganizationId
	result = db.Table("janus_auth_user").Save(&user)
	if result.Error != nil {
		log.Fatal("Failed to update user record, err:", result.Error)
		return
	}
}

func DeleteUserRecord(userName string) {
	db, err := ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer CloseDatabaseConnection(db)
	var user User
	result := db.Table("janus_auth_user").Where("user_name = ?", userName).First(&user)
	if result.Error != nil {
		log.Fatal("Failed to get user record, err:", result.Error)
		return
	}
	result = db.Table("janus_auth_user").Delete(&user)
	if result.Error != nil {
		log.Fatal("Failed to delete user record, err:", result.Error)
		return
	}
}
