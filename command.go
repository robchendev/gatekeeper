package main

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
)

var VerificationRole = discord.RoleID(mustSnowflakeEnv("VERIFIED_ROLE_ID"))
var EmailDomain = mustEnv("EMAIL_DOMAIN")

func Register(email string) (string, error) {
	if err := validateEmail(email); err != nil {
		return err.Error(), nil
	}

	db.VerifiedEmails.M.Lock()
	_, ok := db.VerifiedEmails.D[email]
	db.VerifiedEmails.M.Unlock()
	if ok {
		return "email is currently claimed by a user", nil
	}

	// create token
	b := make([]byte, 4)
	rand.Read(b)
	token := hex.EncodeToString(b)

	db.EmailTokens.M.Lock()
	db.EmailTokens.D[token] = email
	db.EmailTokens.M.Unlock()

	body := formatRegistrationEmail(token)

	return "A email has been sent to " + email + "\nPlease use /verify <token> to verify your email address.", SendEmail(email, "Gatekeeper Email Verification", body)
}

func formatRegistrationEmail(token string) string {
	return ("Greetings from Gatekeeper!\n\n" +
		"Your verification token is: " + token)
}

func Verify(s *state.State, user discord.UserID, guild discord.GuildID, token string) (msg string, err error) {
	db.EmailTokens.M.Lock()
	email, ok := db.EmailTokens.D[token]
	db.EmailTokens.M.Unlock()
	if !ok {
		return "Sorry, verification failed.", nil
	}

	db.VerifiedEmails.M.Lock()
	oldUser, ok := db.VerifiedEmails.D[email]
	db.VerifiedEmails.M.Unlock()
	if ok {
		db.EmailTokens.M.Lock()
		delete(db.EmailTokens.D, token)
		db.EmailTokens.M.Unlock()

		return "Sorry, this email is already in use by <@" + oldUser.String() + ">. Please contact a moderator to be verified manually .", nil
		// msg = msg + "This email was in use by <@" + oldUser.String() + ">. They will now be unverified."
		// // TODO ==================================================
		// oldMember :=
		// if (s.Member(guild, oldUser))
		// err = s.RemoveRole(guild, oldUser, VerificationRole, api.AuditLogReason("Gatekeeper verification"))
		// if err != nil {
		// 	return "", errors.Wrap(err, "cannot remove role from user")
		// }
	}

	err = s.AddRole(guild, user, VerificationRole, api.AddRoleData{AuditLogReason: api.AuditLogReason("Gatekeeper verification")})
	if err != nil {
		return "", err
	}

	db.EmailTokens.M.Lock()
	delete(db.EmailTokens.D, token)
	db.EmailTokens.M.Unlock()

	db.VerifiedEmails.M.Lock()
	db.VerifiedEmails.D[email] = user
	db.VerifiedEmails.M.Unlock()

	msg += "\nCongrats! You've been verified!"

	return msg, nil
}