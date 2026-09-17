import { expect, test } from "bun:test";

import { defaultModelCapabilityConfig } from "../src/lib/model-capabilities";
import { prepareBackendGenerationTask } from "../src/services/api/generation-task";
import { defaultConfig, type AiConfig } from "../src/stores/use-config-store";

test("逻辑视频任务不会提交模型不支持的旧音频和水印开关", async () => {
    const model = "platform::silent-video";
    const profile = defaultModelCapabilityConfig(undefined, "silent-video");
    profile.video!.generateAudio = { supported: false, default: false };
    profile.video!.watermark = { supported: false, default: false };
    const config: AiConfig = {
        ...defaultConfig,
        model,
        videoModel: model,
        videoGenerateAudio: "true",
        videoWatermark: "true",
        models: [model],
        videoModels: [model],
        channels: [
            {
                id: "platform",
                name: "平台模型",
                baseUrl: "/api",
                apiKey: "system",
                apiFormat: "openai",
                scope: "system",
                models: ["silent-video"],
                modelCosts: [
                    {
                        model: "silent-video",
                        capability: "video",
                        billingMode: "per_second",
                        unitPriceMicrocredits: 1,
                        logicalModelId: "silent-video",
                        logicalCapabilitySpec: {
                            version: 1,
                            capability: "video",
                            options: {
                                videoGenerateAudio: { values: [false] },
                                videoWatermark: { values: [false] },
                            },
                        },
                        capabilityConfig: profile,
                    },
                ],
            },
        ],
    };

    const task = await prepareBackendGenerationTask({ mode: "video", prompt: "夜晚的城市", config });
    const input = task.input as { config: Record<string, unknown>; capabilityOptions: Record<string, unknown> };

    expect(input.config.videoGenerateAudio).toBe("false");
    expect(input.config.videoWatermark).toBe("false");
    expect(input.capabilityOptions.videoGenerateAudio).toBe(false);
    expect(input.capabilityOptions.videoWatermark).toBe(false);
});
