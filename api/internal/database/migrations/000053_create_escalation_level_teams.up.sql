CREATE TABLE escalation_level_teams (
    escalation_level_id UUID NOT NULL REFERENCES escalation_levels(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    PRIMARY KEY (escalation_level_id, team_id)
);
