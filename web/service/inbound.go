package service

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/protocol"
	"x-ui/util/common"
	"x-ui/xray"
)

type InboundService struct {
}

func (s *InboundService) GetInbounds(userId int) ([]*model.Inbound, error) {
	db := database.GetDB()
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Where("user_id = ?", userId).Find(&inbounds).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err := s.attachClients(inbounds); err != nil {
		return nil, err
	}
	return inbounds, nil
}

func (s *InboundService) GetAllInbounds() ([]*model.Inbound, error) {
	db := database.GetDB()
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Find(&inbounds).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err := s.attachClients(inbounds); err != nil {
		return nil, err
	}
	return inbounds, nil
}

func (s *InboundService) checkPortExist(port int, ignoreId int) (bool, error) {
	db := database.GetDB()
	db = db.Model(model.Inbound{}).Where("port = ?", port)
	if ignoreId > 0 {
		db = db.Where("id != ?", ignoreId)
	}
	var count int64
	err := db.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *InboundService) AddInbound(inbound *model.Inbound) error {
	exist, err := s.checkPortExist(inbound.Port, 0)
	if err != nil {
		return err
	}
	if exist {
		return common.NewError("端口已存在:", inbound.Port)
	}
	db := database.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		return s.saveInboundWithClients(tx, inbound)
	})
}

func (s *InboundService) AddInbounds(inbounds []*model.Inbound) error {
	for _, inbound := range inbounds {
		exist, err := s.checkPortExist(inbound.Port, 0)
		if err != nil {
			return err
		}
		if exist {
			return common.NewError("端口已存在:", inbound.Port)
		}
	}

	db := database.GetDB()
	tx := db.Begin()
	var err error
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	for _, inbound := range inbounds {
		if err = s.saveInboundWithClients(tx, inbound); err != nil {
			return err
		}
	}

	return nil
}

func (s *InboundService) DelInbound(id int) error {
	db := database.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("inbound_id = ?", id).Delete(&model.InboundClient{}).Error; err != nil {
			return err
		}
		return tx.Delete(model.Inbound{}, id).Error
	})
}

func (s *InboundService) GetInbound(id int) (*model.Inbound, error) {
	db := database.GetDB()
	inbound := &model.Inbound{}
	err := db.Model(model.Inbound{}).First(inbound, id).Error
	if err != nil {
		return nil, err
	}
	if err := s.attachClients([]*model.Inbound{inbound}); err != nil {
		return nil, err
	}
	return inbound, nil
}

func (s *InboundService) UpdateInbound(inbound *model.Inbound) error {
	exist, err := s.checkPortExist(inbound.Port, inbound.Id)
	if err != nil {
		return err
	}
	if exist {
		return common.NewError("端口已存在:", inbound.Port)
	}

	oldInbound, err := s.GetInbound(inbound.Id)
	if err != nil {
		return err
	}
	oldInbound.Up = inbound.Up
	oldInbound.Down = inbound.Down
	oldInbound.Total = inbound.Total
	oldInbound.Remark = inbound.Remark
	oldInbound.Enable = inbound.Enable
	oldInbound.ExpiryTime = inbound.ExpiryTime
	oldInbound.Listen = inbound.Listen
	oldInbound.Port = inbound.Port
	oldInbound.Protocol = inbound.Protocol
	oldInbound.Settings = inbound.Settings
	oldInbound.StreamSettings = inbound.StreamSettings
	oldInbound.Sniffing = inbound.Sniffing
	oldInbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)

	db := database.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		return s.saveInboundWithClients(tx, oldInbound)
	})
}

func (s *InboundService) AddTraffic(traffics []*xray.Traffic) (err error) {
	if len(traffics) == 0 {
		return nil
	}
	db := database.GetDB()
	db = db.Model(model.Inbound{})
	tx := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	for _, traffic := range traffics {
		if traffic.IsInbound {
			err = tx.Where("tag = ?", traffic.Tag).
				UpdateColumn("up", gorm.Expr("up + ?", traffic.Up)).
				UpdateColumn("down", gorm.Expr("down + ?", traffic.Down)).
				Error
			if err != nil {
				return
			}
		}
	}
	return
}

func (s *InboundService) DisableInvalidInbounds() (int64, error) {
	db := database.GetDB()
	now := time.Now().Unix() * 1000
	result := db.Model(model.Inbound{}).
		Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Update("enable", false)
	err := result.Error
	count := result.RowsAffected
	return count, err
}

func (s *InboundService) saveInboundWithClients(tx *gorm.DB, inbound *model.Inbound) error {
	module, err := protocol.DefaultRegistry().Get(string(inbound.Protocol))
	if err != nil {
		return err
	}
	if inbound, err = module.Migrate(inbound); err != nil {
		return err
	}
	if err := module.Validate(inbound); err != nil {
		return err
	}
	settings, clients, err := splitSettingsClients(inbound.Settings)
	if err != nil {
		return err
	}
	inbound.Settings = settings
	if err := tx.Save(inbound).Error; err != nil {
		return err
	}
	if clients == nil {
		return nil
	}
	if err := tx.Where("inbound_id = ?", inbound.Id).Delete(&model.InboundClient{}).Error; err != nil {
		return err
	}
	for _, client := range clients {
		record := &model.InboundClient{
			InboundId: inbound.Id,
			Email:     clientEmail(client),
			ClientKey: clientKey(client),
			Settings:  client,
			Enable:    true,
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *InboundService) attachClients(inbounds []*model.Inbound) error {
	if len(inbounds) == 0 {
		return nil
	}

	ids := make([]int, 0, len(inbounds))
	index := make(map[int]*model.Inbound, len(inbounds))
	for _, inbound := range inbounds {
		ids = append(ids, inbound.Id)
		index[inbound.Id] = inbound
	}

	db := database.GetDB()
	var clients []model.InboundClient
	if err := db.Where("inbound_id in ?", ids).Order("id asc").Find(&clients).Error; err != nil {
		return err
	}
	grouped := make(map[int][]string)
	for _, client := range clients {
		grouped[client.InboundId] = append(grouped[client.InboundId], client.Settings)
	}
	for inboundId, clientSettings := range grouped {
		inbound := index[inboundId]
		if inbound == nil {
			continue
		}
		settings, err := mergeSettingsClients(inbound.Settings, clientSettings)
		if err != nil {
			return err
		}
		inbound.Settings = settings
	}
	return nil
}

func splitSettingsClients(raw string) (string, []string, error) {
	settings := map[string]interface{}{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			return "", nil, err
		}
	}

	rawClients, ok := settings["clients"].([]interface{})
	if !ok {
		return raw, nil, nil
	}
	clients := make([]string, 0, len(rawClients))
	for _, client := range rawClients {
		data, err := json.Marshal(client)
		if err != nil {
			return "", nil, err
		}
		clients = append(clients, string(data))
	}
	delete(settings, "clients")
	data, err := json.Marshal(settings)
	if err != nil {
		return "", nil, err
	}
	return string(data), clients, nil
}

func mergeSettingsClients(raw string, clientSettings []string) (string, error) {
	settings := map[string]interface{}{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			return "", err
		}
	}
	clients := make([]interface{}, 0, len(clientSettings))
	for _, clientRaw := range clientSettings {
		var client interface{}
		if err := json.Unmarshal([]byte(clientRaw), &client); err != nil {
			return "", err
		}
		clients = append(clients, client)
	}
	settings["clients"] = clients
	data, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func clientEmail(raw string) string {
	client := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &client); err != nil {
		return ""
	}
	email, _ := client["email"].(string)
	return email
}

func clientKey(raw string) string {
	client := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &client); err != nil {
		return raw
	}
	for _, key := range []string{"id", "password", "email"} {
		value, _ := client[key].(string)
		if value != "" {
			return value
		}
	}
	return raw
}
