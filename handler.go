package handler

import (
	"time"

	"github.com/google/go-github/github"
)

// New lambda handler with the provided settings.
func New(manager *Manager, secretTemplate, titleTemplate string, logger *logrus.Logger) func(Team) error {
	return func(team Team) error {
		log := logger.WithFields(logrus.Fields{"team": team.Name})

		// Loop through teams and assume roles/write credentials for
		// all accounts controlled by the team.
		for _, repository := range team.Repositories {
			path, err := NewSecretPath(team.Name, repository.Name, secretTemplate).String()
			if err != nil {
				log.WithFields(logrus.Fields{"repository": account.Name}).Warnf("failed to parse secret path: %s", err)
				continue
			}

			title, err := NewPath(team.Name, repository.Name, command.Path).String()
			if err != nil {
				log.WithFields(logrus.Fields{"repository": account.Name}).Warnf("failed to parse github title: %s", err)
				continue
			}

			// Look for existing keys for the team
			keys, err := manager.ListKeys(repository)
			if err != nil {
				log.WithFields(logrus.Fields{"repository": repository.Name}).Warnf("failed to list github keys: %s", err)
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
				log.WithFields(logrus.Fields{"repository": repository.Name}).Warnf("failed to generate new key pair: %s", err)
				continue
			}

			// Write the new public key to Github
			if _, err = manager.CreateKey(repository, path, public); err != nil {
				log.WithFields(logrus.Fields{"repository": account.Name}).Warnf("failed to create key on github: %s", err)
				continue
			}

			// Write the private key to Secrets manager
			if err := manager.WriteSecret(private, path); err != nil {
				log.WithFields(logrus.Fields{"repository": account.Name}).Warnf("failed to write secret key: %s", err)
				continue
			}

			// Sleep before deleting old key (in case someone has just fetched the old key)
			if oldKey != nil {
				time.Sleep(time.Second * 1)
				if err = manager.DeleteKey(repository, int(*oldKey.ID)); err != nil {
					log.WithFields(logrus.Fields{"repository": account.Name}).Warnf("failed to delete old github key: %d: %s", *oldKey.ID, err)
					continue
				}
			}
		}
		return nil
	}
}
