package handlers

import (
	"github.com/bwmarrin/discordgo"
)

type ResponseType int

const (
	ThinkResponse   ResponseType = iota // NewResponseType Respond with a "Bot is thinking..." message
	EphemeralThink                      // NewResponseType Respond with an ephemeral message saying "Bot is thinking..."
	pendingResponse                     // NewResponseType Respond with a "Bot is responding..." message
	messageResponse                     // MsgResponseType Respond with a message

	followupResponse  // MsgReturnType Send a followup message
	followupEdit      // editResponseType Edit a followup message by providing a [*discordgo.Message]
	ephemeralFollowup // MsgReturnType Respond with an ephemeral followup message

	editMessage             // editResponseType Edit a [*discordgo.Message]
	UpdateFromComponent     // MsgResponseType Update the interaction response message
	EditInteractionResponse // MsgReturnType Edit the interaction response message

	ephemeralResponding // NewResponseType Respond with an ephemeral message saying "Bot is responding..."
	ephemeralContent    // MsgResponseType Respond with an ephemeral message with the provided content

	HelloResponse // newResponseType Respond with a message saying "Hey there! Congratulations, you just executed your first slash command"
)

type NewResponseType func(bot *discordgo.Session, i *discordgo.InteractionCreate)
type newReturnType func(bot *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.Message
type MsgResponseType func(bot *discordgo.Session, i *discordgo.Interaction, content ...any)
type MsgReturnType func(bot *discordgo.Session, i *discordgo.Interaction, content ...any) *discordgo.Message
type editResponseType func(bot *discordgo.Session, i *discordgo.Interaction, message *discordgo.Message, content ...any) *discordgo.Message

var Responses = map[ResponseType]any{
	ThinkResponse: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			ErrorEphemeralResponse(bot, i.Interaction, err)
		}
	}),
	EphemeralThink: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			}})
		if err != nil {
			ErrorEphemeralResponse(bot, i.Interaction, err)
		}
	}),
	pendingResponse: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Note: this isn't documented, but you can use that if you want to.
				// This flag just allows you to create messages visible only for the caller of the command
				// (user who triggered the command)
				//Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Bot is responding...",
			},
		})
		if err != nil {
			Errors[ErrorResponse](bot, i.Interaction, err)
		}
	}),
	messageResponse: MsgResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, message ...any) {
		err := bot.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: message[0].(string),
			},
		})
		if err != nil {
			Errors[ErrorFollowup](bot, i, err)
		}
	}),
	followupResponse: MsgReturnType(func(bot *discordgo.Session, i *discordgo.Interaction, message ...any) *discordgo.Message {
		webhookParams := contentToWebhookParams(message...)

		msg, err := bot.FollowupMessageCreate(i, true, &webhookParams)
		if err != nil {
			Errors[ErrorFollowup](bot, i, err)
		}
		return msg
	}),

	followupEdit: editResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, message *discordgo.Message, content ...any) *discordgo.Message {
		// check if any content is a webhook edit
		webhookEdit := webhookFromContents(content...)

		contentEdit(webhookEdit, message)
		contentEdit(webhookEdit, content...)

		msg, err := bot.FollowupMessageEdit(i, message.Reference().MessageID, webhookEdit)
		if err != nil {
			Errors[ErrorFollowup](bot, i, err)
		}
		return msg
	}),

	ephemeralFollowup: MsgReturnType(func(bot *discordgo.Session, i *discordgo.Interaction, message ...any) *discordgo.Message {
		webhookParams := contentToWebhookParams(message...)

		msg, err := bot.FollowupMessageCreate(i, true, &webhookParams)
		if err != nil {
			Errors[ErrorFollowup](bot, i, err)
		}
		return msg
	}),

	editMessage: editResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, message *discordgo.Message, content ...any) *discordgo.Message {
		// check if any content is a webhook edit
		webhookEdit := webhookFromContents(content...)

		contentEdit(webhookEdit, message)
		contentEdit(webhookEdit, content...)

		msg, err := bot.FollowupMessageEdit(i, message.Reference().MessageID, webhookEdit)
		if err != nil {
			Errors[ErrorFollowup](bot, i, err)
		}
		return msg
	}),

	UpdateFromComponent: MsgResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, content ...any) {
		interactionResponse := discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{},
		}

		responseEdit(interactionResponse.Data, content...)

		err := bot.InteractionRespond(i, &interactionResponse)
		if err != nil {
			Errors[ErrorFollowupEphemeral](bot, i, err)
		}
	}),

	EditInteractionResponse: MsgReturnType(func(bot *discordgo.Session, i *discordgo.Interaction, content ...any) *discordgo.Message {
		// check if any content is a webhook edit
		webhookEdit := webhookFromContents(content...)

		contentEdit(webhookEdit, content...)

		msg, err := bot.InteractionResponseEdit(i, webhookEdit)
		if err != nil {
			Errors[ErrorEphemeral](bot, i, err)
		}
		return msg
	}),

	ephemeralResponding: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Note: this isn't documented, but you can use that if you want to.
				// This flag just allows you to create messages visible only for the caller of the command
				// (user who triggered the command)
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Bot is responding...",
			},
		})
		if err != nil {
			Errors[ErrorResponse](bot, i.Interaction, err)
		}
	}),
	ephemeralContent: MsgResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, message ...any) {
		err := bot.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Note: this isn't documented, but you can use that if you want to.
				// This flag just allows you to create messages visible only for the caller of the command
				// (user who triggered the command)
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: message[0].(string),
			},
		})
		if err != nil {
			Errors[ErrorFollowup](bot, i, err)
		}
	}),
	HelloResponse: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Hey there! Congratulations, you just executed your first slash command",
			},
		})
		if err != nil {
			Errors[ErrorResponse](bot, i.Interaction, err)
		}
	}),
}

func webhookFromContents(content ...any) *discordgo.WebhookEdit {
	webhookEdit := &discordgo.WebhookEdit{}
	for _, m := range content {
		switch c := m.(type) {
		case discordgo.WebhookEdit:
			webhookEdit = &c
		case *discordgo.WebhookEdit:
			webhookEdit = c
		}
	}
	return webhookEdit
}

func contentToWebhookParams(content ...any) discordgo.WebhookParams {
	webhookParams := discordgo.WebhookParams{}
	for _, m := range content {
		switch c := m.(type) {
		case discordgo.WebhookParams:
			webhookParams = c
		case string:
			webhookParams.Content = c
		case discordgo.MessageComponent:
			webhookParams.Components = append(webhookParams.Components, c)
		case discordgo.MessageFlags:
			webhookParams.Flags = c
		}
	}
	return webhookParams
}

func contentEdit(webhookEdit *discordgo.WebhookEdit, messages ...any) {
	if len(messages) == 0 {
		return
	}
	var newEmbeds []*discordgo.MessageEmbed
	var newComponents []discordgo.MessageComponent
	for _, m := range messages {
		switch c := m.(type) {
		case *discordgo.Message:
			webhookEdit.Content = &c.Content
			webhookEdit.Embeds = &c.Embeds
			webhookEdit.Components = &c.Components
		case string:
			//log.Println("String content: ", c)
			webhookEdit.Content = &c
		case discordgo.MessageEmbed:
			newEmbeds = append(newEmbeds, &c)
		case discordgo.MessageComponent:
			newComponents = append(newComponents, c)
		case []discordgo.MessageComponent:
			newComponents = append(newComponents, c...)
		}
	}
	if len(newComponents) > 0 {
		webhookEdit.Components = &newComponents
	}
	if len(newEmbeds) > 0 {
		webhookEdit.Embeds = &newEmbeds
	}
}

func responseEdit(resp *discordgo.InteractionResponseData, messages ...any) {
	if resp == nil {
		resp = &discordgo.InteractionResponseData{}
	}
	if len(messages) == 0 {
		return
	}
	var newEmbeds []*discordgo.MessageEmbed
	var newComponents []discordgo.MessageComponent
	for _, m := range messages {
		switch c := m.(type) {
		case *discordgo.Message:
			resp.Content = c.Content
			resp.Embeds = c.Embeds
			resp.Components = c.Components
		case string:
			resp.Content = c
		case discordgo.MessageEmbed:
			newEmbeds = append(newEmbeds, &c)
		case discordgo.MessageComponent:
			newComponents = append(newComponents, c)
		case []discordgo.MessageComponent:
			newComponents = append(newComponents, c...)
		}
	}
	if len(newComponents) > 0 {
		resp.Components = newComponents
	}
	if len(newEmbeds) > 0 {
		resp.Embeds = newEmbeds
	}
}
func EphemeralFollowup(bot *discordgo.Session, i *discordgo.Interaction, message ...any) {
	Responses[ephemeralFollowup].(MsgReturnType)(bot, i, message...)
}

func DeleteAboveFollowup(bot *discordgo.Session, i *discordgo.Interaction) {
	Errors[ErrorFollowupEphemeral](bot, i, "Delete generation", Components[DeleteButton])
}
