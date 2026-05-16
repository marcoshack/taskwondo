-- TF-350: sub-line anchors and threaded replies for inline comments.
--
-- Migration 000061 added whole-line inline-comment anchors. This migration
-- extends them so a comment can be attached to an arbitrary text selection
-- and so inline comments can carry threaded replies.
--
--  * anchor_start_col / anchor_end_col — 1-based character offsets within the
--    anchor's start/end line (end_col exclusive). NULL/0 for whole-block
--    anchors with no sub-line precision.
--  * parent_comment_id — threads a reply onto a root inline comment. A reply
--    carries a copy of its root's anchor so it survives re-anchoring
--    independently. Root comments (regular or inline) have a NULL parent.
--
-- It is a separate migration (not folded into 000061) because 000061 has
-- already been applied to existing databases.

ALTER TABLE comments
    ADD COLUMN anchor_start_col  INT,
    ADD COLUMN anchor_end_col    INT,
    ADD COLUMN parent_comment_id UUID REFERENCES comments(id);

-- Help thread assembly find a root comment's replies.
CREATE INDEX idx_comments_parent
    ON comments (parent_comment_id)
    WHERE parent_comment_id IS NOT NULL AND deleted_at IS NULL;
