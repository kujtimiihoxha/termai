package tools

type agentTool struct {
	workingDir string
}

const (
	AgentToolName = "agent"
)

type AgentParams struct {
	Prompt string `json:"prompt"`
}

// func taskAgentTools() []tool.BaseTool {
// 	wd := viper.GetString("wd")
// 	return []tool.BaseTool{
// 		NewBashTool(wd),
// 		NewLsTool(wd),
// 		NewGlobTool(wd),
// 		NewGrepTool(wd),
// 		NewViewTool(wd),
// 	}
// }
//
// func NewTaskAgent(ctx context.Context) (*react.Agent, error) {
// 	model, err := models.GetModel(ctx, models.ModelID(viper.GetString("models.big")))
// 	if err != nil {
// 		return nil, err
// 	}
// 	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
// 		Model: model,
// 		ToolsConfig: compose.ToolsNodeConfig{
// 			Tools: taskAgentTools(),
// 		},
// 		MaxStep: 1000,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return reactAgent, nil
// }

// func TaskAgentSystemPrompt() string {
// 	agentPrompt := `You are an agent for Orbitowl. Given the user's prompt, you should use the tools available to you to answer the user's question.
//
// Notes:
// 1. IMPORTANT: You should be concise, direct, and to the point, since your responses will be displayed on a command line interface. Answer the user's question directly, without elaboration, explanation, or details. One word answers are best. Avoid introductions, conclusions, and explanations. You MUST avoid text before/after your response, such as "The answer is <answer>.", "Here is the content of the file..." or "Based on the information provided, the answer is..." or "Here is what I will do next...".
// 2. When relevant, share file names and code snippets relevant to the query
// 3. Any file paths you return in your final response MUST be absolute. DO NOT use relative paths.
//
// Here is useful information about the environment you are running in:
// <env>
// Working directory: %s
// Platform: %s
// Today's date: %s
// </env>`
//
// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		cwd = "unknown"
// 	}
//
// 	platform := runtime.GOOS
//
// 	switch platform {
// 	case "darwin":
// 		platform = "macos"
// 	case "windows":
// 		platform = "windows"
// 	case "linux":
// 		platform = "linux"
// 	}
// 	return fmt.Sprintf(agentPrompt, cwd, platform, time.Now().Format("1/2/2006"))
// }
//
// func (b *agentTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
// 	return &schema.ToolInfo{
// 		Name: AgentToolName,
// 		Desc: "Launch a new agent that has access to the following tools: GlobTool, GrepTool, LS, View. When you are searching for a keyword or file and are not confident that you will find the right match on the first try, use the Agent tool to perform the search for you. For example:\n\n- If you are searching for a keyword like \"config\" or \"logger\", or for questions like \"which file does X?\", the Agent tool is strongly recommended\n- If you want to read a specific file path, use the View or GlobTool tool instead of the Agent tool, to find the match more quickly\n- If you are searching for a specific class definition like \"class Foo\", use the GlobTool tool instead, to find the match more quickly\n\nUsage notes:\n1. Launch multiple agents concurrently whenever possible, to maximize performance; to do that, use a single message with multiple tool uses\n2. When the agent is done, it will return a single message back to you. The result returned by the agent is not visible to the user. To show the user the result, you should send a text message back to the user with a concise summary of the result.\n3. Each agent invocation is stateless. You will not be able to send additional messages to the agent, nor will the agent be able to communicate with you outside of its final report. Therefore, your prompt should contain a highly detailed task description for the agent to perform autonomously and you should specify exactly what information the agent should return back to you in its final and only message to you.\n4. The agent's outputs should generally be trusted\n5. IMPORTANT: The agent can not use Bash, Replace, Edit, NotebookEditCell, so can not modify files. If you want to use these tools, use them directly instead of going through the agent.",
// 		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
// 			"prompt": {
// 				Type:     "string",
// 				Desc:     "The task for the agent to perform",
// 				Required: true,
// 			},
// 		}),
// 	}, nil
// }
//
// func (b *agentTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
// 	var params AgentParams
// 	if err := json.Unmarshal([]byte(args), &params); err != nil {
// 		return "", err
// 	}
// 	if params.Prompt == "" {
// 		return "prompt is required", nil
// 	}
//
// 	a, err := NewTaskAgent(ctx)
// 	if err != nil {
// 		return "", err
// 	}
// 	out, err := a.Generate(
// 		ctx,
// 		[]*schema.Message{
// 			schema.SystemMessage(TaskAgentSystemPrompt()),
// 			schema.UserMessage(params.Prompt),
// 		},
// 	)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	return out.Content, nil
// }
//
// func NewAgentTool(wd string) tool.InvokableTool {
// 	return &agentTool{
// 		workingDir: wd,
// 	}
// }
