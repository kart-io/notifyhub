# 智能路由增强方案

## 概述

智能路由增强方案通过引入AI驱动的路由决策、自适应配置和机器学习优化，将NotifyHub的路由系统从基于规则的静态路由升级为智能、自适应的动态路由系统。

## 当前路由系统分析

### 现有路由机制

```go
// 当前基于规则的路由
type Rule struct {
    Name       string     `json:"name"`
    Priority   int        `json:"priority"`
    Enabled    bool       `json:"enabled"`
    Conditions Conditions `json:"conditions"`
    Actions    Actions    `json:"actions"`
}

type Conditions struct {
    Priorities []int               `json:"priorities,omitempty"`
    Metadata   map[string]string   `json:"metadata,omitempty"`
    Platforms  []string           `json:"platforms,omitempty"`
}
```

### 局限性分析

1. **静态规则**：无法根据实时情况动态调整
2. **缺乏学习能力**：无法从历史数据中学习优化
3. **简单条件匹配**：无法处理复杂的业务逻辑
4. **无自适应能力**：无法根据平台状态自动调整

## 智能路由设计方案

### 1. 智能路由核心架构

```go
// 智能路由器接口
type IntelligentRouter interface {
    // 路由决策
    RouteMessage(ctx context.Context, msg *message.Message, options RoutingOptions) (*RoutingDecision, error)
    BatchRouteMessages(ctx context.Context, messages []*message.Message, options RoutingOptions) ([]*RoutingDecision, error)

    // 学习和优化
    LearnFromResult(decision *RoutingDecision, result *sending.SendingResults) error
    TrainModel(trainingData []RoutingTrainingData) error

    // 策略管理
    GetRoutingStrategies() []RoutingStrategy
    SetRoutingStrategy(strategy RoutingStrategy) error
    UpdateStrategyWeights(weights map[string]float64) error

    // 分析和预测
    AnalyzeRoutingPerformance(timeRange TimeRange) *RoutingAnalysis
    PredictOptimalRoute(msg *message.Message, timeWindow time.Duration) *RoutePrediction
    GetRoutingRecommendations(ctx context.Context) []RoutingRecommendation
}

// 路由决策
type RoutingDecision struct {
    MessageID       string                 `json:"message_id"`
    Targets         []sending.Target       `json:"targets"`
    PrimaryPlatform string                 `json:"primary_platform"`
    BackupPlatforms []string               `json:"backup_platforms"`
    Confidence      float64                `json:"confidence"`      // 决策置信度
    Reasoning       []string               `json:"reasoning"`       // 决策理由
    Metadata        map[string]interface{} `json:"metadata"`
    Timestamp       time.Time              `json:"timestamp"`
    Strategy        string                 `json:"strategy"`        // 使用的策略
    Alternatives    []AlternativeRoute     `json:"alternatives"`    // 备选路由
}

// 备选路由
type AlternativeRoute struct {
    Targets    []sending.Target `json:"targets"`
    Platform   string           `json:"platform"`
    Score      float64          `json:"score"`
    Reasoning  string           `json:"reasoning"`
    Fallback   bool             `json:"fallback"`
}

// 路由选项
type RoutingOptions struct {
    PreferredPlatforms []string               `json:"preferred_platforms"`
    ExcludedPlatforms  []string               `json:"excluded_platforms"`
    Requirements       RoutingRequirements    `json:"requirements"`
    Context            map[string]interface{} `json:"context"`
    EnableFallback     bool                   `json:"enable_fallback"`
    MaxAlternatives    int                    `json:"max_alternatives"`
}

type RoutingRequirements struct {
    MinReliability     float64       `json:"min_reliability"`     // 最小可靠性
    MaxLatency         time.Duration `json:"max_latency"`         // 最大延迟
    RequireDeliveryConf bool         `json:"require_delivery_conf"` // 需要送达确认
    CostConstraint     float64       `json:"cost_constraint"`     // 成本约束
    ComplianceLevel    string        `json:"compliance_level"`    // 合规级别
}
```

### 2. 机器学习路由引擎

```go
// ML路由引擎
type MLRoutingEngine struct {
    models          map[string]MLModel
    featureExtractor *FeatureExtractor
    dataCollector   *RoutingDataCollector
    predictor       *RoutingPredictor
    config          MLRoutingConfig
}

type MLRoutingConfig struct {
    EnableMLRouting     bool          `json:"enable_ml_routing"`
    ModelUpdateInterval time.Duration `json:"model_update_interval"`
    TrainingDataSize    int           `json:"training_data_size"`
    ConfidenceThreshold float64       `json:"confidence_threshold"`
    FallbackToRules     bool          `json:"fallback_to_rules"`
    Models              []ModelConfig `json:"models"`
}

type ModelConfig struct {
    Name        string                 `json:"name"`
    Type        string                 `json:"type"`        // "decision_tree", "neural_network", "ensemble"
    Enabled     bool                   `json:"enabled"`
    Weight      float64                `json:"weight"`
    Parameters  map[string]interface{} `json:"parameters"`
    UpdateFreq  time.Duration          `json:"update_freq"`
}

// ML模型接口
type MLModel interface {
    Train(data []RoutingTrainingData) error
    Predict(features FeatureVector) (*RoutingPrediction, error)
    UpdateWithFeedback(prediction *RoutingPrediction, actual *RoutingResult) error
    GetModelMetrics() ModelMetrics
    Save(path string) error
    Load(path string) error
}

// 特征提取器
type FeatureExtractor struct {
    extractors map[string]FeatureExtractorFunc
    normalizer *FeatureNormalizer
}

type FeatureExtractorFunc func(*message.Message, context.Context) (interface{}, error)

type FeatureVector struct {
    MessageFeatures  MessageFeatures  `json:"message_features"`
    ContextFeatures  ContextFeatures  `json:"context_features"`
    PlatformFeatures PlatformFeatures `json:"platform_features"`
    TemporalFeatures TemporalFeatures `json:"temporal_features"`
}

type MessageFeatures struct {
    Priority        int                    `json:"priority"`
    BodyLength      int                    `json:"body_length"`
    TitleLength     int                    `json:"title_length"`
    HasTemplate     bool                   `json:"has_template"`
    VariableCount   int                    `json:"variable_count"`
    MetadataCount   int                    `json:"metadata_count"`
    MessageType     string                 `json:"message_type"`
    Urgency         float64                `json:"urgency"`
    ContentCategory string                 `json:"content_category"`
    Language        string                 `json:"language"`
    Sentiment       float64                `json:"sentiment"`
}

type ContextFeatures struct {
    TimeOfDay       int     `json:"time_of_day"`        // 0-23
    DayOfWeek       int     `json:"day_of_week"`        // 0-6
    IsHoliday       bool    `json:"is_holiday"`
    IsBusinessHour  bool    `json:"is_business_hour"`
    UserTimezone    string  `json:"user_timezone"`
    SystemLoad      float64 `json:"system_load"`
    QueueSize       int     `json:"queue_size"`
    RecentErrorRate float64 `json:"recent_error_rate"`
}

type PlatformFeatures struct {
    AvailablePlatforms  []string             `json:"available_platforms"`
    PlatformReliability map[string]float64   `json:"platform_reliability"`
    PlatformLatency     map[string]float64   `json:"platform_latency"`
    PlatformCost        map[string]float64   `json:"platform_cost"`
    RateLimitStatus     map[string]float64   `json:"rate_limit_status"`
    HealthScores        map[string]float64   `json:"health_scores"`
}

type TemporalFeatures struct {
    RecentSuccessRate    float64 `json:"recent_success_rate"`
    RecentAverageLatency float64 `json:"recent_average_latency"`
    TrendDirection       string  `json:"trend_direction"`     // "improving", "degrading", "stable"
    SeasonalPattern      string  `json:"seasonal_pattern"`
    PeakTrafficTime      bool    `json:"peak_traffic_time"`
}

// 决策树模型实现
type DecisionTreeModel struct {
    tree      *DecisionTree
    maxDepth  int
    minSamples int
    features  []string
    metrics   ModelMetrics
}

func (dtm *DecisionTreeModel) Train(data []RoutingTrainingData) error {
    // 准备训练数据
    features := make([]FeatureVector, len(data))
    labels := make([]string, len(data))

    for i, sample := range data {
        features[i] = sample.Features
        labels[i] = sample.OptimalPlatform
    }

    // 构建决策树
    dtm.tree = BuildDecisionTree(features, labels, dtm.maxDepth, dtm.minSamples)

    // 计算模型指标
    dtm.metrics = dtm.evaluateModel(features, labels)

    return nil
}

func (dtm *DecisionTreeModel) Predict(features FeatureVector) (*RoutingPrediction, error) {
    if dtm.tree == nil {
        return nil, fmt.Errorf("model not trained")
    }

    result := dtm.tree.Predict(features)

    prediction := &RoutingPrediction{
        Platform:   result.Platform,
        Confidence: result.Confidence,
        Reasoning:  result.Path, // 决策路径
        Score:      result.Score,
    }

    return prediction, nil
}

// 神经网络模型实现
type NeuralNetworkModel struct {
    network   *NeuralNetwork
    optimizer *Optimizer
    config    NeuralNetworkConfig
    metrics   ModelMetrics
}

type NeuralNetworkConfig struct {
    HiddenLayers []int   `json:"hidden_layers"`
    LearningRate float64 `json:"learning_rate"`
    BatchSize    int     `json:"batch_size"`
    Epochs       int     `json:"epochs"`
    Dropout      float64 `json:"dropout"`
}

func (nnm *NeuralNetworkModel) Train(data []RoutingTrainingData) error {
    // 准备训练数据
    inputs, outputs := nnm.prepareTrainingData(data)

    // 训练神经网络
    for epoch := 0; epoch < nnm.config.Epochs; epoch++ {
        for i := 0; i < len(inputs); i += nnm.config.BatchSize {
            end := i + nnm.config.BatchSize
            if end > len(inputs) {
                end = len(inputs)
            }

            batchInputs := inputs[i:end]
            batchOutputs := outputs[i:end]

            // 前向传播
            predictions := nnm.network.Forward(batchInputs)

            // 计算损失
            loss := nnm.calculateLoss(predictions, batchOutputs)

            // 反向传播
            gradients := nnm.network.Backward(loss)

            // 更新权重
            nnm.optimizer.Update(nnm.network, gradients)
        }
    }

    return nil
}

// 集成模型
type EnsembleModel struct {
    models   []MLModel
    weights  []float64
    strategy EnsembleStrategy
    metrics  ModelMetrics
}

type EnsembleStrategy string

const (
    EnsembleStrategyVoting    EnsembleStrategy = "voting"
    EnsembleStrategyWeighted  EnsembleStrategy = "weighted"
    EnsembleStrategyStacking  EnsembleStrategy = "stacking"
)

func (em *EnsembleModel) Predict(features FeatureVector) (*RoutingPrediction, error) {
    predictions := make([]*RoutingPrediction, len(em.models))

    // 获取各模型预测
    for i, model := range em.models {
        pred, err := model.Predict(features)
        if err != nil {
            continue
        }
        predictions[i] = pred
    }

    // 根据策略合并预测结果
    switch em.strategy {
    case EnsembleStrategyVoting:
        return em.votingPredict(predictions), nil
    case EnsembleStrategyWeighted:
        return em.weightedPredict(predictions), nil
    case EnsembleStrategyStacking:
        return em.stackingPredict(predictions), nil
    default:
        return em.votingPredict(predictions), nil
    }
}
```

### 3. 自适应路由策略

```go
// 自适应路由策略
type AdaptiveRoutingStrategy struct {
    strategyManager *StrategyManager
    adaptationEngine *AdaptationEngine
    performanceMonitor *PerformanceMonitor
    config AdaptiveConfig
}

type AdaptiveConfig struct {
    AdaptationInterval    time.Duration `json:"adaptation_interval"`
    PerformanceThreshold  float64       `json:"performance_threshold"`
    MinAdaptationSamples  int           `json:"min_adaptation_samples"`
    MaxStrategies         int           `json:"max_strategies"`
    EnableAutoCreation    bool          `json:"enable_auto_creation"`
    LearningRate          float64       `json:"learning_rate"`
}

// 策略管理器
type StrategyManager struct {
    strategies    map[string]*RoutingStrategy
    activeStrategy string
    performance   map[string]*StrategyPerformance
    optimizer     *StrategyOptimizer
}

type RoutingStrategy struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Rules       []AdaptiveRule         `json:"rules"`
    Weights     map[string]float64     `json:"weights"`
    Conditions  []StrategyCondition    `json:"conditions"`
    Active      bool                   `json:"active"`
    Created     time.Time              `json:"created"`
    LastUsed    time.Time              `json:"last_used"`
    Performance *StrategyPerformance   `json:"performance"`
}

type AdaptiveRule struct {
    ID          string      `json:"id"`
    Condition   string      `json:"condition"`    // 可执行的条件表达式
    Action      string      `json:"action"`       // 路由动作
    Weight      float64     `json:"weight"`       // 规则权重
    Confidence  float64     `json:"confidence"`   // 规则置信度
    Usage       int         `json:"usage"`        // 使用次数
    Success     int         `json:"success"`      // 成功次数
    Adaptive    bool        `json:"adaptive"`     // 是否自适应
}

type StrategyCondition struct {
    Type      string      `json:"type"`       // "time", "load", "performance", "custom"
    Operator  string      `json:"operator"`   // "gt", "lt", "eq", "in", "match"
    Value     interface{} `json:"value"`
    Threshold float64     `json:"threshold"`
}

type StrategyPerformance struct {
    SuccessRate     float64       `json:"success_rate"`
    AverageLatency  time.Duration `json:"average_latency"`
    Throughput      float64       `json:"throughput"`
    ErrorRate       float64       `json:"error_rate"`
    CostEfficiency  float64       `json:"cost_efficiency"`
    UserSatisfaction float64      `json:"user_satisfaction"`
    LastEvaluated   time.Time     `json:"last_evaluated"`
    SampleSize      int           `json:"sample_size"`
}

// 自适应引擎
type AdaptationEngine struct {
    learningAlgorithm LearningAlgorithm
    strategyGenerator *StrategyGenerator
    evaluator         *StrategyEvaluator
    config            AdaptationConfig
}

type LearningAlgorithm interface {
    Learn(experiences []RoutingExperience) error
    GenerateStrategy(context RoutingContext) (*RoutingStrategy, error)
    OptimizeStrategy(strategy *RoutingStrategy, feedback []RoutingFeedback) (*RoutingStrategy, error)
}

// 强化学习算法实现
type ReinforcementLearning struct {
    qTable    map[string]map[string]float64 // Q表
    epsilon   float64                       // 探索率
    alpha     float64                       // 学习率
    gamma     float64                       // 折扣因子
    actions   []string                      // 可能的动作（平台）
    states    []string                      // 可能的状态
}

func (rl *ReinforcementLearning) Learn(experiences []RoutingExperience) error {
    for _, exp := range experiences {
        state := rl.encodeState(exp.Context)
        action := exp.Action
        reward := rl.calculateReward(exp.Result)
        nextState := rl.encodeState(exp.NextContext)

        // Q-Learning更新
        currentQ := rl.getQValue(state, action)
        maxNextQ := rl.getMaxQValue(nextState)
        newQ := currentQ + rl.alpha*(reward+rl.gamma*maxNextQ-currentQ)

        rl.setQValue(state, action, newQ)
    }

    return nil
}

func (rl *ReinforcementLearning) GenerateStrategy(context RoutingContext) (*RoutingStrategy, error) {
    state := rl.encodeState(context)

    // ε-贪心策略选择
    if rand.Float64() < rl.epsilon {
        // 探索：随机选择
        action := rl.actions[rand.Intn(len(rl.actions))]
        return rl.createStrategyFromAction(action, context), nil
    } else {
        // 利用：选择最优动作
        bestAction := rl.getBestAction(state)
        return rl.createStrategyFromAction(bestAction, context), nil
    }
}

// 遗传算法实现
type GeneticAlgorithm struct {
    population     []RoutingStrategy
    populationSize int
    mutationRate   float64
    crossoverRate  float64
    generations    int
    fitness        map[string]float64
}

func (ga *GeneticAlgorithm) Learn(experiences []RoutingExperience) error {
    // 评估当前种群适应度
    ga.evaluateFitness(experiences)

    for generation := 0; generation < ga.generations; generation++ {
        // 选择
        selected := ga.selection()

        // 交叉
        offspring := ga.crossover(selected)

        // 变异
        ga.mutation(offspring)

        // 更新种群
        ga.population = ga.updatePopulation(offspring)

        // 重新评估
        ga.evaluateFitness(experiences)
    }

    return nil
}

func (ga *GeneticAlgorithm) crossover(parents []RoutingStrategy) []RoutingStrategy {
    var offspring []RoutingStrategy

    for i := 0; i < len(parents)-1; i += 2 {
        if rand.Float64() < ga.crossoverRate {
            child1, child2 := ga.crossoverTwoParents(parents[i], parents[i+1])
            offspring = append(offspring, child1, child2)
        } else {
            offspring = append(offspring, parents[i], parents[i+1])
        }
    }

    return offspring
}
```

### 4. 实时性能监控和预测

```go
// 性能监控器
type PerformanceMonitor struct {
    metrics     *MetricsCollector
    predictor   *PerformancePredictor
    alertManager *AlertManager
    dashboard   *PerformanceDashboard
}

type MetricsCollector struct {
    routingMetrics   map[string]*RoutingMetrics
    platformMetrics  map[string]*PlatformMetrics
    systemMetrics    *SystemMetrics
    timeSeriesDB     TimeSeriesDatabase
}

type RoutingMetrics struct {
    TotalRequests    int64         `json:"total_requests"`
    SuccessfulRoutes int64         `json:"successful_routes"`
    FailedRoutes     int64         `json:"failed_routes"`
    AverageLatency   time.Duration `json:"average_latency"`
    P95Latency       time.Duration `json:"p95_latency"`
    P99Latency       time.Duration `json:"p99_latency"`
    Throughput       float64       `json:"throughput"`
    ErrorRate        float64       `json:"error_rate"`
    ConfidenceScore  float64       `json:"confidence_score"`
    LastUpdated      time.Time     `json:"last_updated"`
}

// 性能预测器
type PerformancePredictor struct {
    models       map[string]PredictionModel
    dataStore    *HistoricalDataStore
    features     *FeatureStore
}

type PredictionModel interface {
    PredictPerformance(features FeatureVector, timeHorizon time.Duration) (*PerformancePrediction, error)
    PredictPlatformLoad(platform string, timeHorizon time.Duration) (*LoadPrediction, error)
    PredictOptimalRouting(context RoutingContext) (*RoutingPrediction, error)
}

type PerformancePrediction struct {
    Metric      string        `json:"metric"`
    Prediction  float64       `json:"prediction"`
    Confidence  float64       `json:"confidence"`
    TimeHorizon time.Duration `json:"time_horizon"`
    Factors     []string      `json:"factors"`
    Trend       string        `json:"trend"`
}

// 时间序列预测模型
type TimeSeriesModel struct {
    model     *ARIMA
    seasonal  bool
    trend     bool
    config    TimeSeriesConfig
}

type TimeSeriesConfig struct {
    WindowSize    int     `json:"window_size"`
    Seasonality   int     `json:"seasonality"`   // 季节性周期
    TrendDamping  float64 `json:"trend_damping"`
    SeasonDamping float64 `json:"season_damping"`
    Confidence    float64 `json:"confidence"`
}

func (tsm *TimeSeriesModel) PredictPerformance(features FeatureVector, timeHorizon time.Duration) (*PerformancePrediction, error) {
    // 提取时间序列数据
    timeSeries := tsm.extractTimeSeries(features)

    // 预测未来值
    forecast := tsm.model.Forecast(timeSeries, int(timeHorizon/time.Minute))

    // 计算置信区间
    confidence := tsm.calculateConfidence(forecast)

    prediction := &PerformancePrediction{
        Prediction:  forecast.Value,
        Confidence:  confidence,
        TimeHorizon: timeHorizon,
        Trend:       tsm.analyzeTrend(forecast),
    }

    return prediction, nil
}
```

### 5. 智能路由配置和管理

```go
// 智能路由配置管理器
type IntelligentRoutingManager struct {
    router           IntelligentRouter
    configStore      ConfigurationStore
    strategyManager  *StrategyManager
    experimentManager *ExperimentManager
    optimizer        *RoutingOptimizer
}

// A/B测试管理器
type ExperimentManager struct {
    experiments    map[string]*RoutingExperiment
    trafficSplitter *TrafficSplitter
    analyzer       *ExperimentAnalyzer
}

type RoutingExperiment struct {
    ID          string           `json:"id"`
    Name        string           `json:"name"`
    Description string           `json:"description"`
    Strategies  []string         `json:"strategies"`      // 参与实验的策略
    TrafficSplit map[string]float64 `json:"traffic_split"` // 流量分配
    StartTime   time.Time        `json:"start_time"`
    EndTime     *time.Time       `json:"end_time"`
    Status      ExperimentStatus `json:"status"`
    Results     *ExperimentResults `json:"results"`
    Hypothesis  string           `json:"hypothesis"`
    Metrics     []string         `json:"metrics"`         // 关注指标
}

type ExperimentStatus string

const (
    ExperimentStatusPlanning ExperimentStatus = "planning"
    ExperimentStatusRunning  ExperimentStatus = "running"
    ExperimentStatusPaused   ExperimentStatus = "paused"
    ExperimentStatusCompleted ExperimentStatus = "completed"
    ExperimentStatusFailed   ExperimentStatus = "failed"
)

type ExperimentResults struct {
    Winner       string                 `json:"winner"`
    Confidence   float64                `json:"confidence"`
    Significance float64                `json:"significance"`
    Metrics      map[string]float64     `json:"metrics"`
    Insights     []string               `json:"insights"`
    Recommendations []string            `json:"recommendations"`
}

// 流量分割器
type TrafficSplitter struct {
    algorithms map[string]SplitAlgorithm
    config     TrafficSplitConfig
}

type SplitAlgorithm interface {
    ShouldRoute(userID string, experimentID string, split float64) bool
    GetVariant(userID string, experimentID string, variants []string, weights []float64) string
}

// 路由优化器
type RoutingOptimizer struct {
    optimizer    OptimizationAlgorithm
    constraints  []OptimizationConstraint
    objectives   []OptimizationObjective
}

type OptimizationAlgorithm interface {
    Optimize(problem OptimizationProblem) (*OptimizationSolution, error)
}

type OptimizationProblem struct {
    Variables   []OptimizationVariable   `json:"variables"`
    Constraints []OptimizationConstraint `json:"constraints"`
    Objectives  []OptimizationObjective  `json:"objectives"`
    Context     map[string]interface{}   `json:"context"`
}

type OptimizationSolution struct {
    Variables []VariableValue `json:"variables"`
    Objective float64         `json:"objective"`
    Status    string          `json:"status"`
    Runtime   time.Duration   `json:"runtime"`
}

// 粒子群优化算法实现
type ParticleSwarmOptimization struct {
    particles      []Particle
    swarmSize      int
    maxIterations  int
    inertiaWeight  float64
    cognitive      float64
    social         float64
    globalBest     *Particle
}

type Particle struct {
    Position     []float64 `json:"position"`
    Velocity     []float64 `json:"velocity"`
    PersonalBest []float64 `json:"personal_best"`
    Fitness      float64   `json:"fitness"`
}

func (pso *ParticleSwarmOptimization) Optimize(problem OptimizationProblem) (*OptimizationSolution, error) {
    // 初始化粒子群
    pso.initializeSwarm(problem)

    for iteration := 0; iteration < pso.maxIterations; iteration++ {
        // 更新粒子位置和速度
        for i := range pso.particles {
            pso.updateParticle(&pso.particles[i], iteration)
        }

        // 评估适应度
        pso.evaluateFitness(problem)

        // 更新全局最优
        pso.updateGlobalBest()
    }

    return pso.createSolution(), nil
}
```

## 集成和使用

### 1. 智能路由配置

```go
// 配置智能路由
func WithIntelligentRouting(config IntelligentRoutingConfig) Option {
    return func(cfg *Config) {
        cfg.IntelligentRouting = &config
    }
}

type IntelligentRoutingConfig struct {
    Enabled              bool                   `json:"enabled"`
    MLConfig             MLRoutingConfig        `json:"ml_config"`
    AdaptiveConfig       AdaptiveConfig         `json:"adaptive_config"`
    ExperimentConfig     ExperimentConfig       `json:"experiment_config"`
    OptimizationConfig   OptimizationConfig     `json:"optimization_config"`
    FallbackToRules      bool                   `json:"fallback_to_rules"`
    TrainingDataRetention time.Duration         `json:"training_data_retention"`
}

// 使用示例
client, err := notifyhub.New(
    notifyhub.WithFeishu("webhook", "secret"),
    notifyhub.WithEmail("smtp.server.com", 587, "user", "pass", "from"),
    notifyhub.WithIntelligentRouting(IntelligentRoutingConfig{
        Enabled: true,
        MLConfig: MLRoutingConfig{
            EnableMLRouting:     true,
            ModelUpdateInterval: 1 * time.Hour,
            ConfidenceThreshold: 0.8,
            Models: []ModelConfig{
                {
                    Name:    "decision_tree",
                    Type:    "decision_tree",
                    Enabled: true,
                    Weight:  0.4,
                },
                {
                    Name:    "neural_network",
                    Type:    "neural_network",
                    Enabled: true,
                    Weight:  0.6,
                },
            },
        },
        AdaptiveConfig: AdaptiveConfig{
            AdaptationInterval:   30 * time.Minute,
            PerformanceThreshold: 0.95,
            EnableAutoCreation:   true,
        },
        FallbackToRules: true,
    }),
)
```

### 2. 智能路由使用

```go
// 使用智能路由发送消息
result, err := client.Send(ctx).
    Title("智能路由测试").
    Body("系统将自动选择最优平台").
    Priority(message.PriorityHigh).
    EnableIntelligentRouting().
    Execute()

// 获取路由决策信息
if intelligentResult, ok := result.(*IntelligentRoutingResult); ok {
    log.Printf("选择的平台: %s", intelligentResult.Decision.PrimaryPlatform)
    log.Printf("决策置信度: %.2f", intelligentResult.Decision.Confidence)
    log.Printf("决策理由: %v", intelligentResult.Decision.Reasoning)
}
```

### 3. 路由策略管理

```go
// 创建自定义路由策略
strategy := &RoutingStrategy{
    Name: "high_priority_strategy",
    Rules: []AdaptiveRule{
        {
            Condition: "message.Priority >= 4 && context.IsBusinessHour",
            Action:    "route_to_primary_platform",
            Weight:    0.8,
        },
        {
            Condition: "platform.ErrorRate < 0.01",
            Action:    "prefer_reliable_platform",
            Weight:    0.9,
        },
    },
}

// 注册策略
err := client.GetIntelligentRouter().GetStrategyManager().AddStrategy(strategy)

// 启动A/B测试
experiment := &RoutingExperiment{
    Name: "Strategy Comparison",
    Strategies: []string{"current_strategy", "high_priority_strategy"},
    TrafficSplit: map[string]float64{
        "current_strategy":      0.5,
        "high_priority_strategy": 0.5,
    },
    Metrics: []string{"success_rate", "latency", "user_satisfaction"},
}

err = client.GetExperimentManager().StartExperiment(experiment)
```

### 4. 性能监控和分析

```go
// 获取路由性能分析
analysis := client.GetIntelligentRouter().AnalyzeRoutingPerformance(TimeRange{
    Start: time.Now().Add(-24 * time.Hour),
    End:   time.Now(),
})

log.Printf("总路由请求: %d", analysis.TotalRequests)
log.Printf("成功率: %.2f%%", analysis.SuccessRate*100)
log.Printf("平均延迟: %v", analysis.AverageLatency)

// 获取路由建议
recommendations := client.GetIntelligentRouter().GetRoutingRecommendations(ctx)
for _, rec := range recommendations {
    log.Printf("建议: %s (置信度: %.2f)", rec.Description, rec.Confidence)
}

// 预测最优路由
prediction := client.GetIntelligentRouter().PredictOptimalRoute(message, 1*time.Hour)
log.Printf("预测最优平台: %s", prediction.Platform)
```

## 效果预期

### 性能提升指标

| 指标 | 基础路由 | 智能路由 | 提升幅度 |
|------|---------|---------|----------|
| **路由准确率** | 70% | 95% | +35% |
| **平均延迟** | 200ms | 120ms | -40% |
| **成功率** | 92% | 98% | +6% |
| **成本效率** | 基准 | 基准×1.3 | +30% |
| **自适应速度** | 手动 | 自动30min | +∞ |

### 业务价值

1. **用户体验提升**：更快、更可靠的消息送达
2. **运营成本降低**：智能选择成本最优的平台
3. **运维效率提升**：自动化路由优化，减少人工干预
4. **系统可靠性**：智能故障转移和负载均衡
5. **业务洞察**：深入的路由性能分析和预测

## 总结

智能路由增强方案通过机器学习、自适应算法和性能预测等技术，将NotifyHub的路由系统升级为智能化、自适应的高性能路由引擎。该方案不仅显著提升了系统性能，还为运营团队提供了强大的分析和优化工具，为未来的智能化运营奠定了坚实基础。