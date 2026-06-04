CREATE TABLE sudoku_puzzles (
    id UUID PRIMARY KEY,
    difficulty VARCHAR(20) NOT NULL,
    puzzle_grid JSONB NOT NULL,
    solution_grid JSONB NOT NULL,
    empty_count INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
