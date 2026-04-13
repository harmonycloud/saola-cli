package consts

// LabelDefinition is the label key that records the baseline name on a CR.
// zeus-operator/pkg/service/consts does not export this constant yet;
// once it does, replace all usages with the upstream definition.
//
// LabelDefinition 记录 CR 所使用的 baseline 名称的 label key。
// zeus-operator/pkg/service/consts 中暂未导出该常量，等上游补充后统一替换。
const LabelDefinition = "middleware.cn/definition"
