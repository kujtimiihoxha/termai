package prompt

func TitlePrompt() string {
	return `you will generate a short title based on the first message a user begins a conversation with
- ensure it is not more than 50 characters long
- the title should be a summary of the user's message
- do not use quotes or colons
- the entire text you return will be used as the title`
}
