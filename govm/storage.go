package govm

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (m *Manager) readLocalData() error {
	filename := filepath.Join(m.workspace, "local.json")
	content, err := os.ReadFile(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		content = []byte("{}")
	}
	var data = new(LocalData)
	if err := json.Unmarshal(content, data); err != nil {
		return err
	}
	m.Data = data
	return nil
}

func (m *Manager) writeLocalData() error {
	filename := filepath.Join(m.workspace, "local.json")
	content, err := json.Marshal(m.Data)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, content, 0600)
}

func (m *Manager) readLocalVersions() (Versions, error) {
	filename := filepath.Join(m.workspace, "versions.json")
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var versions Versions
	if err := json.Unmarshal(content, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func (m *Manager) writeLocalVersions() error {
	filename := filepath.Join(m.workspace, "versions.json")
	content, err := json.Marshal(m.Versions)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, content, 0600)
}

func (m *Manager) walkInstalledVersions() error {
	dir := filepath.Join(m.workspace, "versions")
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	m.Data.InstalledVersions = nil

	for _, entry := range dirEntries {
		if entry.IsDir() {
			m.Data.InstalledVersions = append(m.Data.InstalledVersions, entry.Name())
		}
	}
	return nil
}

func (m *Manager) saveAll() error {
	if err := m.writeLocalData(); err != nil {
		return err
	}
	return m.writeLocalVersions()
}
