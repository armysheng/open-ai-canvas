import { afterAll, describe, expect, test } from "bun:test";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, join, resolve } from "node:path";
import { build as viteBuild } from "vite";

let outputDir: string | undefined;
let buildPromise: Promise<void> | undefined;

async function buildProductionOutput() {
    if (!buildPromise) {
        buildPromise = (async () => {
            outputDir = await mkdtemp(join(tmpdir(), "yingce-welcome-production-"));
            await viteBuild({
                configFile: resolve(import.meta.dir, "../vite.config.ts"),
                logLevel: "silent",
                publicDir: false,
                build: {
                    outDir: outputDir,
                    emptyOutDir: true,
                    reportCompressedSize: false,
                },
            });
        })();
    }

    await buildPromise;
    return outputDir!;
}

async function readMainChunk() {
    const dir = await buildProductionOutput();
    const html = await readFile(join(dir, "index.html"), "utf8");
    const entry = html.match(/<script[^>]+src="\/assets\/([^"?]+\.js)"/);
    if (!entry) throw new Error("Production index.html does not reference an entry script");
    return readFile(join(dir, "assets", entry[1]), "utf8");
}

function welcomePreloadDependencies(mainChunk: string) {
    const dependencyTable = mainChunk.match(/m\.f\|\|\(m\.f=(\[[^\]]*\])\)/)?.[1];
    if (!dependencyTable) throw new Error("Vite entry chunk does not expose its dynamic import dependency table");

    const dependencies = JSON.parse(dependencyTable) as string[];
    const dynamicImport = mainChunk.match(/import\(`\.\/(welcome-application-[^`]+\.js)`\),__vite__mapDeps\(\[([^\]]*)\]\)/);
    if (!dynamicImport) throw new Error("Vite entry chunk does not expose the welcome dynamic import");

    return dynamicImport[2]
        .split(",")
        .filter(Boolean)
        .map((index) => dependencies[Number(index)])
        .filter((dependency): dependency is string => Boolean(dependency))
        .map((dependency) => basename(dependency));
}

afterAll(async () => {
    if (outputDir) await rm(outputDir, { recursive: true, force: true });
});

describe("welcome production chunking", () => {
    test("does not preload the workspace application chunk", async () => {
        const dependencies = welcomePreloadDependencies(await readMainChunk());

        expect(dependencies.some((dependency) => /^application-[^/]+\.js$/.test(dependency))).toBe(false);
    });
});
