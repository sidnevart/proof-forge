import { describe, expect, it } from "vitest";

import { metadata } from "./layout";

describe("app layout metadata", () => {
  it("uses russian product description", () => {
    expect(metadata.title).toBe("ProofForge");
    expect(metadata.description).toBe(
      "Система внешней ответственности для серьёзных целей и подтверждённого прогресса.",
    );
  });
});
