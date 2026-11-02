-- Existing blocks used a 15-minute duration. Preserve their start time,
-- shortening them to 5 minutes without resetting the failure counters.
UPDATE auth_limits
SET blocked_until = MAX(0, blocked_until - 600)
WHERE blocked_until > 0;
