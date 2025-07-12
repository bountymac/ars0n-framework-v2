-- Add screenshot_path column to consolidated_attack_surface_assets table
ALTER TABLE consolidated_attack_surface_assets ADD COLUMN IF NOT EXISTS screenshot_path TEXT; 