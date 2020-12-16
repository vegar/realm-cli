package realm_test

import (
	"testing"

	"github.com/10gen/realm-cli/internal/cloud/realm"
	u "github.com/10gen/realm-cli/internal/utils/test"
	"github.com/10gen/realm-cli/internal/utils/test/assert"
)

func TestRealmUsers(t *testing.T) {
	u.SkipUnlessRealmServerRunning(t)

	client := newAuthClient(t)
	groupID := u.CloudGroupID()

	app, appErr := client.CreateApp(groupID, "users-test", realm.AppMeta{})
	assert.Nil(t, appErr)

	t.Run("Should fail without an auth client", func(t *testing.T) {
		badClient := realm.NewClient(u.RealmServerURL())

		_, err := badClient.FindUsers(groupID, app.ID, realm.UserFilter{})
		assert.Equal(t, realm.ErrInvalidSession, err)
	})

	t.Run("With an active session ", func(t *testing.T) {
		// need to enable local-userpass and api-key auth providers first
		t.Skip("TODO(REALMC-7632): implement basic version of import to facilitate test setup")

		t.Run("Should create users", func(t *testing.T) {
			email1, createErr := client.CreateUser(groupID, app.ID, "one@domain.com", "password1")
			assert.Nil(t, createErr)
			email2, createErr := client.CreateUser(groupID, app.ID, "two@domain.com", "password2")
			assert.Nil(t, createErr)
			email3, createErr := client.CreateUser(groupID, app.ID, "three@domain.com", "password3")
			assert.Nil(t, createErr)

			apiKey1, createErr := client.CreateAPIKey(groupID, app.ID, "one")
			assert.Nil(t, createErr)
			apiKey2, createErr := client.CreateAPIKey(groupID, app.ID, "two")
			assert.Nil(t, createErr)

			apiKeyIDs := map[string]struct{}{
				apiKey1.ID: struct{}{},
				apiKey2.ID: struct{}{},
			}

			t.Run("And find all types of users", func(t *testing.T) {
				users, err := client.FindUsers(groupID, app.ID, realm.UserFilter{})
				assert.Nil(t, err)

				emailUsers := make([]realm.User, 0, 3)
				apiKeyUsers := make([]realm.User, 0, 2)

				for _, user := range users {
					switch user.Type {
					case "local-userpass":
						emailUsers = append(emailUsers, user)
					case "api-key":
						apiKeyUsers = append(apiKeyUsers, user)
					}
				}

				assert.Equal(t, 3, len(emailUsers))
				assert.Equal(t, []realm.User{email1, email2, email3}, emailUsers)

				assert.Equal(t, 2, len(apiKeyUsers))
				for _, user := range apiKeyUsers {
					assert.Equalf(t, 1, len(user.Identities), "expected api key user to have one identity")

					identity := user.Identities[0]
					_, ok := apiKeyIDs[identity.UID]
					assert.True(t, ok, "expected %s to match a previously created api key id", identity.UID)
				}
			})

			t.Run("And find a certain type of user", func(t *testing.T) {
				users, err := client.FindUsers(groupID, app.ID, realm.UserFilter{Providers: []string{"local-userpass"}})
				assert.Nil(t, err)
				assert.Equal(t, []realm.User{email1, email2, email3}, users)
			})

			t.Run("And find specific user ids", func(t *testing.T) {
				users, err := client.FindUsers(groupID, app.ID, realm.UserFilter{IDs: []string{email2.ID, email3.ID}})
				assert.Nil(t, err)
				assert.Equal(t, []realm.User{email2, email3}, users)
			})

			t.Run("And disable users", func(t *testing.T) {
				assert.Nil(t, client.DisableUser(groupID, app.ID, email1.ID))
				assert.Nil(t, client.DisableUser(groupID, app.ID, email3.ID))
			})

			t.Run("And find all disabled users", func(t *testing.T) {
				users, err := client.FindUsers(groupID, app.ID, realm.UserFilter{State: realm.UserStateDisabled})
				assert.Nil(t, err)
				assert.Equal(t, []realm.User{email1, email3}, users)
			})

			t.Run("And find specific user using all filter options", func(t *testing.T) {
				// target email3
				filter := realm.UserFilter{
					IDs:       []string{email2.ID, email3.ID, apiKey1.ID},
					State:     realm.UserStateDisabled,
					Providers: []string{"local-userpass"},
				}
				users, err := client.FindUsers(groupID, app.ID, filter)
				assert.Nil(t, err)
				assert.Equal(t, []realm.User{email3}, users)
			})

			t.Run("And delete users", func(t *testing.T) {
				for _, userID := range []string{email1.ID, email2.ID, email3.ID, apiKey1.ID, apiKey2.ID} {
					assert.Nilf(t, client.DeleteUser(groupID, app.ID, userID), "failed to successfully delete user: %s", userID)
				}
			})

			t.Run("And revoking a user session should succeed", func(t *testing.T) {
				assert.Nil(t, client.RevokeUserSessions(groupID, app.ID, email1.ID))
			})
		})

		t.Run("And finding pending users should return an empty list", func(t *testing.T) {
			users, err := client.FindUsers(groupID, app.ID, realm.UserFilter{Pending: true})
			assert.Nil(t, err)
			assert.Equal(t, []realm.User{}, users)
		})
	})
}
