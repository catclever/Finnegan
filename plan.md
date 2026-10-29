1. Fix Vision Cropping Logic
   - The user requested that we detect bounds and crop the image. Gemini Vision natively allows you to ask for bounding boxes. The format often uses `[ymin, xmin, ymax, xmax]`. I'll update the prompt to ask for a `box` (array of integers `[y_min, x_min, y_max, x_max]` scaled to 1000). Then use the `image` library to crop it.
2. Fix YAML Frontmatter Construction
   - Instead of writing raw strings, use `yaml.Marshal` for the metadata to ensure it's valid YAML.
3. Link Command to IngestScreenshot
   - The `ExecuteCommand` for `curio:ingest` needs to parse `screenshot` and `note` and actually call `IngestScreenshot`.
4. Add YAML Serialization and Fallback Parsing Unit Tests
   - As requested by the user, add explicit tests for frontmatter serialization and fallback parsing in `modules/curio/vision_test.go` or `cards_test.go`/`storage_test.go`. The user asked for "unit tests in modules/curio/ testing YAML frontmatter serialization, slug sanitization, and fallback parsing." I'll add them to `modules/curio/vision_test.go`.
5. Pre-commit & Submit
