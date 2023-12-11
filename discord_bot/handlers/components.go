package handlers

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

const (
	CheckpointSelect   Component = "imagine_sd_model_name_menu"
	VAESelect          Component = "imagine_vae_model_name_menu"
	HypernetworkSelect Component = "imagine_hypernetwork_model_name_menu"
	DimensionSelect    Component = "imagine_dimension_setting_menu"
	BatchCountSelect   Component = "imagine_batch_count_setting_menu"
	BatchSizeSelect    Component = "imagine_batch_size_setting_menu"
)

const (
	RerollButton  Component = "imagine_reroll"
	UpscaleButton Component = "imagine_upscale"
	VariantButton Component = "imagine_variation"
)

type Component string

const (
	DeleteButton      Component = "delete_error_message"
	DeleteAboveButton Component = "delete_above"
	DeleteGeneration  Component = "delete_generation"

	dismissButton Component = "dismiss_error_message"
	urlButton     Component = "url_button"
	urlDelete     Component = "url_delete"

	readmoreDismiss Component = "readmore_dismiss"

	paginationButtons Component = "pagination_button"
	okCancelButtons   Component = "ok_cancel_buttons"

	Cancel    Component = "cancel"
	Interrupt Component = "interrupt"

	CancelDisabled    Component = "cancel_disabled"
	InterruptDisabled Component = "interrupt_disabled"

	JSONInput Component = "raw"

	roleSelect = "role_select"
)

var minValues = 1

var Components = map[Component]discordgo.MessageComponent{
	DeleteButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete this message",
				Style:    discordgo.DangerButton,
				CustomID: string(DeleteButton),
			},
		},
	},
	DeleteAboveButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete above",
				Style:    discordgo.DangerButton,
				CustomID: string(DeleteAboveButton),
				Emoji: discordgo.ComponentEmoji{
					Name: "🗑️",
				},
			},
		},
	},
	DeleteGeneration: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete",
				Style:    discordgo.DangerButton,
				CustomID: string(DeleteGeneration),
				Emoji: discordgo.ComponentEmoji{
					Name: "🗑️",
				},
			},
		},
	},
	urlButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label: "Read more",
				Style: discordgo.LinkButton,
			},
		},
	},
	urlDelete: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label: "Read more",
				Style: discordgo.LinkButton,
				Emoji: discordgo.ComponentEmoji{
					Name: "📜",
				},
			},
			discordgo.Button{
				Label:    "Delete",
				Style:    discordgo.DangerButton,
				CustomID: string(DeleteButton),
				Emoji: discordgo.ComponentEmoji{
					Name: "🗑️",
				},
			},
		},
	},
	dismissButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Dismiss",
				Style:    discordgo.SecondaryButton,
				CustomID: string(DeleteButton),
			},
		},
	},
	readmoreDismiss: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Read more",
				Style:    discordgo.LinkButton,
				CustomID: string(urlButton),
			},
			discordgo.Button{
				Label:    "Dismiss",
				Style:    discordgo.SecondaryButton,
				CustomID: string(DeleteButton),
			},
		},
	},

	paginationButtons: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: string(paginationButtons + "_previous"),
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: string(paginationButtons + "_next"),
			},
		},
	},
	okCancelButtons: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "OK",
				Style:    discordgo.SuccessButton,
				CustomID: string(okCancelButtons + "_ok"),
			},
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.DangerButton,
				CustomID: string(okCancelButtons + "_cancel"),
			},
		},
	},
	roleSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.RoleSelectMenu,
				CustomID:    roleSelect,
				Placeholder: "Pick a role",
			},
		},
	},
	Cancel: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.DangerButton,
				CustomID: string(Cancel),
			},
		},
	},
	CancelDisabled: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.DangerButton,
				CustomID: string(Cancel),
				Disabled: true,
			},
		},
	},
	Interrupt: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Interrupt",
				Style:    discordgo.DangerButton,
				CustomID: string(Interrupt),
				Emoji: discordgo.ComponentEmoji{
					Name: "⚠️",
				},
				Disabled: false,
			},
		},
	},
	InterruptDisabled: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Interrupt",
				Style:    discordgo.DangerButton,
				CustomID: string(Interrupt),
				Emoji: discordgo.ComponentEmoji{
					Name: "⚠️",
				},
				Disabled: true,
			},
		},
	},

	CheckpointSelect:   ModelSelectMenu(CheckpointSelect),
	VAESelect:          ModelSelectMenu(VAESelect),
	HypernetworkSelect: ModelSelectMenu(HypernetworkSelect),

	DimensionSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  string(DimensionSelect),
				MinValues: nil,
				MaxValues: 1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:   "Size: 512x512",
						Value:   "512_512",
						Default: true,
					},
					{
						Label:   "Size: 768x768",
						Value:   "768_768",
						Default: false,
					},
				},
			},
		},
	},
	BatchCountSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  string(BatchCountSelect),
				MinValues: &minValues,
				MaxValues: 1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:   "Batch count: 1",
						Value:   "1",
						Default: false,
					},
					{
						Label:   "Batch count: 2",
						Value:   "2",
						Default: false,
					},
					{
						Label:   "Batch count: 4",
						Value:   "4",
						Default: true,
					},
				},
			},
		},
	},
	BatchSizeSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  string(BatchSizeSelect),
				MinValues: &minValues,
				MaxValues: 1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:   "Batch size: 1",
						Value:   "1",
						Default: true,
					},
					{
						Label:   "Batch size: 2",
						Value:   "2",
						Default: false,
					},
					{
						Label:   "Batch size: 4",
						Value:   "4",
						Default: false,
					},
				},
			},
		},
	},

	JSONInput: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    string(JSONInput),
				Label:       "JSON blob",
				Style:       discordgo.TextInputParagraph,
				Placeholder: "{\"height\":768,\"width\":512,\"prompt\":\"masterpiece\"}",
				Value:       "",
				Required:    true,
				MinLength:   1,
				MaxLength:   4000,
			},
		},
	},
}

func ModelSelectMenu(ID Component) discordgo.ActionsRow {
	display := strings.TrimPrefix(string(ID), "imagine_")
	display = strings.TrimSuffix(string(ID), "_model_name_menu")
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    string(ID),
				Placeholder: fmt.Sprintf("Change %s Model", display),
				MinValues:   &minValues,
				MaxValues:   1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:       display,
						Value:       "Placeholder",
						Description: "Placeholder",
						Default:     false,
					},
				},
			},
		},
	}
}
