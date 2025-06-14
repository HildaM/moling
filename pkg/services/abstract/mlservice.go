package abstract

import (
	"context"
	"sync"

	"github.com/gojue/moling/pkg/comm"
	"github.com/gojue/moling/pkg/config"
	"github.com/gojue/moling/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
)

type PromptEntry struct {
	PromptVar   mcp.Prompt
	HandlerFunc server.PromptHandlerFunc
}

func (pe *PromptEntry) Prompt() mcp.Prompt {
	return pe.PromptVar
}

func (pe *PromptEntry) Handler() server.PromptHandlerFunc {
	return pe.HandlerFunc
}

// MLService implements the Service interface and provides methods to manage resources, templates, prompts, tools, and notification handlers.
type MLService struct {
	Context              context.Context
	lock                 *sync.Mutex
	resources            map[mcp.Resource]server.ResourceHandlerFunc
	resourcesTemplates   map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc
	prompts              []PromptEntry
	tools                []server.ServerTool
	notificationHandlers map[string]server.NotificationHandlerFunc
	Logger               zerolog.Logger       // The logger for the service
	mlConfig             *config.MoLingConfig // The configuration for the service
}

// NewMLService creates a new MLService with the given context and logger.
func NewMLService(ctx context.Context, logger zerolog.Logger, cfg *config.MoLingConfig) MLService {
	service := MLService{
		Context:              ctx,
		Logger:               logger,
		mlConfig:             cfg,
		lock:                 &sync.Mutex{},
		resources:            make(map[mcp.Resource]server.ResourceHandlerFunc),
		resourcesTemplates:   make(map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc),
		prompts:              make([]PromptEntry, 0),
		notificationHandlers: make(map[string]server.NotificationHandlerFunc),
		tools:                []server.ServerTool{},
	}
	return service
}

// init initializes the MLService with empty maps and a mutex.
func (mls *MLService) InitResources() error {
	if mls.lock == nil {
		mls.lock = &sync.Mutex{}
	}
	if mls.resources == nil {
		mls.resources = make(map[mcp.Resource]server.ResourceHandlerFunc)
	}
	if mls.resourcesTemplates == nil {
		mls.resourcesTemplates = make(map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc)
	}
	if mls.prompts == nil {
		mls.prompts = make([]PromptEntry, 0)
	}
	if mls.notificationHandlers == nil {
		mls.notificationHandlers = make(map[string]server.NotificationHandlerFunc)
	}
	if mls.tools == nil {
		mls.tools = []server.ServerTool{}
	}
	return nil
}

// Ctx returns the context of the MLService.
func (mls *MLService) Ctx() context.Context {
	return mls.Context
}

// AddResource adds a resource and its handler function to the service.
func (mls *MLService) AddResource(rs mcp.Resource, hr server.ResourceHandlerFunc) {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	mls.resources[rs] = hr
}

// AddResourceTemplate adds a resource template and its handler function to the service.
func (mls *MLService) AddResourceTemplate(rt mcp.ResourceTemplate, hr server.ResourceTemplateHandlerFunc) {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	mls.resourcesTemplates[rt] = hr
}

// AddPrompt adds a prompt and its handler function to the service.
func (mls *MLService) AddPrompt(pe PromptEntry) {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	mls.prompts = append(mls.prompts, pe)
}

// AddTool adds a tool and its handler function to the service.
func (mls *MLService) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	mls.tools = append(mls.tools, server.ServerTool{Tool: tool, Handler: handler})
}

// AddNotificationHandler adds a notification handler to the service.
func (mls *MLService) AddNotificationHandler(name string, handler server.NotificationHandlerFunc) {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	mls.notificationHandlers[name] = handler
}

// Resources returns the map of resources and their handler functions.
func (mls *MLService) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	return mls.resources
}

// ResourceTemplates returns the map of resource templates and their handler functions.
func (mls *MLService) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	return mls.resourcesTemplates
}

// Prompts returns the map of prompts and their handler functions.
func (mls *MLService) Prompts() []PromptEntry {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	return mls.prompts
}

// Tools returns the slice of server tools.
func (mls *MLService) Tools() []server.ServerTool {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	return mls.tools
}

// NotificationHandlers returns the map of notification handlers.
func (mls *MLService) NotificationHandlers() map[string]server.NotificationHandlerFunc {
	mls.lock.Lock()
	defer mls.lock.Unlock()
	return mls.notificationHandlers
}

// MlConfig returns the configuration of the MoLing service.
func (mls *MLService) MlConfig() *config.MoLingConfig {
	return mls.mlConfig
}

// Config returns the configuration of the service as a string.
func (mls *MLService) Config() string {
	return mls.mlConfig.String()
}

// Name returns the name of the service.
func (mls *MLService) Name() comm.MoLingServerType {
	return "MLService"
}

// LoadConfig loads the configuration for the service from a map.
func (mls *MLService) LoadConfig(jsonData map[string]interface{}) error {
	//panic("not implemented yet") // TODO: Implement
	err := utils.MergeJSONToStruct(mls.MlConfig(), jsonData)
	if err != nil {
		return err
	}
	return mls.MlConfig().Check()
}
