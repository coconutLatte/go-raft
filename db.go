package go_raft

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
)

const (
	dbDir  = "/var/lib/raft/db"
	dbFile = "raft.db"
)

type DB struct {
	base *gorm.DB
}

func OpenDB(num int) (db *DB, err error) {
	err = os.MkdirAll(fmt.Sprintf("%s%d", dbDir, num), 0755)
	if err != nil {
		return
	}

	dsn := filepath.Join(fmt.Sprintf("%s%d", dbDir, num), dbFile)

	base, err := gorm.Open(sqlite.Open(dsn))
	if err != nil {
		return
	}

	err = base.Migrator().AutoMigrate(&NodeInfo{}, &LeaderNode{})
	if err != nil {
		return
	}

	db = &DB{
		base: base,
	}

	return
}

type NodeInfo struct {
	ID      int64  `gorm:"not null;primaryKey;autoIncrement"`
	Address string `gorm:"unique"`
	Role    Role
}

func (NodeInfo) TableName() string {
	return "node_infos"
}

func (d *DB) CreateNodeInfos(nodes []NodeInfo) error {
	return d.base.Create(&nodes).Error
}

func (d *DB) UpdateNodeRole(address string, role Role) error {
	return d.base.
		Model(&NodeInfo{}).
		Where("address = ?", address).
		Update("role", role).
		Error
}

func (d *DB) GetNodeInfos() []NodeInfo {
	var nodeInfos []NodeInfo
	d.base.Find(&nodeInfos).Order("id")

	return nodeInfos
}

func (d *DB) GetNodeInfo(address string) (*NodeInfo, error) {
	var nodeInfo *NodeInfo
	result := d.base.Where("address = ?", address).First(&nodeInfo)
	if result.Error != nil {
		return nil, result.Error
	}

	return nodeInfo, nil
}

type LeaderNode struct {
	Address     string `gorm:"uniqueIndex"`
	LeaderRound int
}

func (LeaderNode) TableName() string {
	return "leader_node"
}

func (d *DB) CreateLeaderNode(ln *LeaderNode) error {
	return d.base.Create(ln).Error
}

func (d *DB) GetLeaderNode() (*LeaderNode, error) {
	var leaderNodes []LeaderNode
	d.base.Find(&leaderNodes)

	if len(leaderNodes) == 0 {
		return nil, fmt.Errorf("leader not exist in db")
	}

	return &leaderNodes[0], nil
}

func (d *DB) UpdateLeaderNode(leaderNode *LeaderNode) error {
	ln, err := d.GetLeaderNode()
	if err != nil {
		return err
	}

	return d.base.
		Model(&LeaderNode{}).
		Where("address = ?", ln.Address).
		Updates(leaderNode).
		Error
}

func (d *DB) IncreaseLeaderRound() error {
	ln, err := d.GetLeaderNode()
	if err != nil {
		return err
	}

	return d.base.
		Model(&LeaderNode{}).
		Where("address = ?", ln.Address).
		Update("leader_round", ln.LeaderRound+1).
		Error
}
