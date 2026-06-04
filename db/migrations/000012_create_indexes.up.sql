CREATE INDEX idx_match_players_user ON match_players(user_id);
CREATE INDEX idx_match_moves_match ON match_moves(match_id);
CREATE INDEX idx_wallet_transactions_user ON wallet_transactions(user_id);
CREATE INDEX idx_user_rank ON users(rank_tier, rank_point DESC);
