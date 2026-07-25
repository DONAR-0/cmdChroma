package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	HFRepo      string `json:"hf_repo"`
	Description string `json:"description"`
}

type Registry struct {
	Models []ModelInfo `json:"models"`
}

type ModelManager struct {
	logger       *slog.Logger
	modelsDir    string
	registryPath string
	mu           sync.RWMutex
	installed    map[string]bool
}

func NewModelManager(logger *slog.Logger, modelsDir string) *ModelManager {
	return &ModelManager{
		logger:       logger,
		modelsDir:    modelsDir,
		registryPath: filepath.Join(modelsDir, "registry.json"),
		installed:    make(map[string]bool),
	}
}

func (m *ModelManager) LoadInstalled() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := os.ReadDir(m.modelsDir)
	if err != nil {
		return fmt.Errorf("failed to read models dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			m.installed[entry.Name()] = true
		}
	}

	return nil
}

func (m *ModelManager) GetAvailableModels() ([]ModelInfo, error) {
	data, err := os.ReadFile(m.registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return reg.Models, nil
}

func (m *ModelManager) IsInstalled(modelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.installed[modelID]
}

func (m *ModelManager) DownloadModel(modelID string, progressFn func(int)) error {
	models, err := m.GetAvailableModels()
	if err != nil {
		return err
	}

	var target ModelInfo

	for _, mod := range models {
		if mod.ID == modelID {
			target = mod
			break
		}
	}

	if target.ID == "" {
		return fmt.Errorf("model %s not found in registry", modelID)
	}

	m.logger.Info("downloading model", "model", modelID, "repo", target.HFRepo)

	dest := filepath.Join(m.modelsDir, modelID)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create model dir: %w", err)
	}

	// In a real implementation, we'd use huggingface-hub or a custom downloader
	// For this MVP, we'll use a shell command to simulate the download or use git-lfs
	// We assume huggingface-cli is installed on the system.
	cmd := exec.Command("huggingface-cli", "download", target.HFRepo, "--local-dir", dest, "--local-dir-use-symlinks", "false")

	// Note: progressFn is tricky with exec.Command. We'd need to parse stdout.
	// For now, we just run the command.
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	m.mu.Lock()
	m.installed[modelID] = true
	m.mu.Unlock()

	m.logger.Info("download completed", "model", modelID)

	return nil
}

func (m *ModelManager) GetModelPath(modelID string) (string, error) {
	if !m.IsInstalled(modelID) {
		return "", fmt.Errorf("model %s not installed", modelID)
	}

	return filepath.Join(m.modelsDir, modelID), nil
}
