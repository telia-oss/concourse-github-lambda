package handler

import (
	"time"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
)

// New lambda handler with the provided settings.
func New(manager *Manager, secretTemplate, titleTemplate string, logger *logrus.Logger) func(Team) error {
	return func(team Team) error {
		for _, repository := range team.Repositories {
			log := logger.WithFields(logrus.Fields{
				"team":       team.Name,
				"repository": repository.Name,
				"owner":      repository.Owner,
			})
			path, err := NewTemplate(team.Name, repository.Name, secretTemplate).String()
			if err != nil {
				log.Warnf("failed to parse secret path: %s", err)
				continue
			}

			title, err := NewTemplate(team.Name, repository.Name, titleTemplate).String()
			if err != nil {
				log.Warnf("failed to parse github title: %s", err)
				continue
			}

			// Look for existing keys for the team
			keys, err := manager.ListKeys(repository)
			if err != nil {
				log.Warnf("failed to list github keys: %s", err)
				continue
			}

			var oldKey *github.Key
			for _, key := range keys {
				if *key.Title == title {
					oldKey = key
				}
			}

			// Generate a new key pair
			private, public, err := manager.GenerateKeyPair(title)
			if err != nil {
				log.Warnf("failed to generate new key pair: %s", err)
				continue
			}

			// Write the new public key to Github
			if err = manager.CreateKey(repository, title, public); err != nil {
				log.Warnf("failed to create key on github: %s", err)
				continue
			}

			// Write the private key to Secrets manager
			if err := manager.WriteSecret(path, private); err != nil {
				log.Warnf("failed to write secret key: %s", err)
				continue
			}

			// Sleep before deleting old key (in case someone has just fetched the old key)
			if oldKey != nil {
				time.Sleep(time.Second * 1)
				if err = manager.DeleteKey(repository, int(*oldKey.ID)); err != nil {
					log.Warnf("failed to delete old github key: %d: %s", *oldKey.ID, err)
					continue
				}
			}
		}
		return nil
	}
}
