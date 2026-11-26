import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { HarnessModelsPanel } from "./HarnessModelsPanel";

const h = vi.hoisted(() => ({
	catalog: undefined as {
		models: Array<{ id: string; label: string; isDefault?: boolean }>;
		allowCustom?: boolean;
		customModelEntry?: "direct" | "configured" | "none";
	} | undefined,
	harnessDefaults: {} as Record<string, { disabledModels?: string[]; defaultModel?: string; hidden?: boolean }>,
}));
afterEach(() => {
	h.catalog = undefined;
	h.harnessDefaults = {};
	update.mockClear();
});

const update = vi.fn(async (_input: { agentId: string; disabledModels?: string[]; defaultModel?: string; hidden?: boolean }) => undefined);
vi.mock("../../../hooks/useSettings", () => ({
	useSettings: () => ({ settings: { harnessDefaults: h.harnessDefaults } }),
	useUpdateHarnessDefault: () => ({ update, error: undefined }),
}));
vi.mock("../../../hooks/useAgentModelsQuery", () => ({
	agentModelsQueryOptions: () => ({
		queryKey: ["agent-models", "cursor"],
		queryFn: async () => h.catalog ?? ({
			models: [
				{ id: "claude-opus-4-8-high", label: "Claude Opus 4.8 High", isDefault: true },
				{ id: "claude-opus-4-8-thinking-high", label: "Claude Opus 4.8 High Thinking" },
				{ id: "gpt-5.5-high", label: "GPT-5.5 High" },
			],
		}),
	}),
}));

describe("HarnessModelsPanel", () => {
	it("explains Mastra Code model-ID entry when its native catalog is freeform", async () => {
		h.catalog = { models: [], allowCustom: true, customModelEntry: "direct" };
		render(
			<QueryClientProvider client={new QueryClient()}>
				<HarnessModelsPanel agentId="mastra-code" installed needsLogin={false} onOpenAccount={() => {}} />
			</QueryClientProvider>,
		);
		expect(await screen.findByText(/Enter a model ID in New Task/)).toBeInTheDocument();
		expect(screen.queryByText("This harness reported no models.")).not.toBeInTheDocument();
	});

	it.each([
		["mastra-code", "configured"], ["mastra-code", "none"], ["codex", "direct"],
	] as const)("does not advertise model-ID entry for %s with a %s catalog", async (agentId, customModelEntry) => {
		h.catalog = { models: [], allowCustom: true, customModelEntry };
		render(
			<QueryClientProvider client={new QueryClient()}>
				<HarnessModelsPanel agentId={agentId} installed needsLogin={false} onOpenAccount={() => {}} />
			</QueryClientProvider>,
		);
		expect(await screen.findByText("This harness reported no models.")).toBeInTheDocument();
		expect(screen.queryByText(/Enter a model ID in New Task/)).not.toBeInTheDocument();
	});

	it("switches a Cursor model's variants together", async () => {
		render(
			<QueryClientProvider client={new QueryClient()}>
				<HarnessModelsPanel agentId="cursor" installed needsLogin={false} onOpenAccount={() => {}} />
			</QueryClientProvider>,
		);
		const opus = await screen.findByRole("switch", { name: /Claude Opus 4.8/ });
		expect(screen.getAllByRole("switch")).toHaveLength(2);
		await userEvent.click(opus);
		expect(update).toHaveBeenCalledWith({ agentId: "cursor", disabledModels: ["claude-opus-4-8-high", "claude-opus-4-8-thinking-high"] });
	});

	it("notes when the harness is hidden and updates disabledModels without sending hidden", async () => {
		h.harnessDefaults = { cursor: { hidden: true } };
		render(
			<QueryClientProvider client={new QueryClient()}>
				<HarnessModelsPanel agentId="cursor" installed needsLogin={false} onOpenAccount={() => {}} />
			</QueryClientProvider>,
		);
		expect(await screen.findByText("Cursor is hidden from pickers. Existing sessions are unaffected.")).toBeInTheDocument();
		const opus = await screen.findByRole("switch", { name: /Claude Opus 4.8/ });
		await userEvent.click(opus);
		expect(update).toHaveBeenCalledWith({ agentId: "cursor", disabledModels: ["claude-opus-4-8-high", "claude-opus-4-8-thinking-high"] });
		expect(update.mock.calls[0][0]).not.toHaveProperty("hidden");
	});

	it("sets a custom default model, re-enables it if disabled, and resets back to CLI native default", async () => {
		h.catalog = {
			models: [
				{ id: "gpt-a", label: "GPT A", isDefault: true },
				{ id: "gpt-b", label: "GPT B" },
			],
		};
		h.harnessDefaults = { codex: { disabledModels: ["gpt-b"] } };
		const qc = new QueryClient();
		const { rerender } = render(
			<QueryClientProvider client={qc}>
				<HarnessModelsPanel agentId="codex" installed needsLogin={false} onOpenAccount={() => {}} />
			</QueryClientProvider>,
		);
		// CLI native default shows "Default" without a "Set as default" button.
		expect(await screen.findByText("Default")).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Set GPT A as default" })).not.toBeInTheDocument();
		// Setting a disabled model as default re-enables it in the same PATCH.
		await userEvent.click(screen.getByRole("button", { name: "Set GPT B as default" }));
		expect(update).toHaveBeenCalledWith({ agentId: "codex", defaultModel: "gpt-b", disabledModels: [] });
		// Once settings reflect the saved default, Reset default restores CLI native default.
		h.harnessDefaults = { codex: { disabledModels: [], defaultModel: "gpt-b" } };
		rerender(
			<QueryClientProvider client={qc}>
				<HarnessModelsPanel agentId="codex" installed needsLogin={false} onOpenAccount={() => {}} />
			</QueryClientProvider>,
		);
		const resetButtons = await screen.findAllByRole("button", { name: "Reset default model for Codex" });
		await userEvent.click(resetButtons[0]);
		expect(update).toHaveBeenLastCalledWith({ agentId: "codex", defaultModel: "" });
	});

	it("clears custom defaultModel when disabling that model", async () => {
		h.catalog = {
			models: [
				{ id: "gpt-a", label: "GPT A", isDefault: true },
				{ id: "gpt-b", label: "GPT B" },
			],
		};
		h.harnessDefaults = { codex: { defaultModel: "gpt-b" } };
		render(
			<QueryClientProvider client={new QueryClient()}>
				<HarnessModelsPanel agentId="codex" installed needsLogin={false} onOpenAccount={() => {}} />
			</QueryClientProvider>,
		);
		const toggleB = await screen.findByRole("switch", { name: "Show GPT B in model pickers" });
		await userEvent.click(toggleB);
		expect(update).toHaveBeenCalledWith({ agentId: "codex", disabledModels: ["gpt-b"], defaultModel: "" });
	});
});
