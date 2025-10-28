-- Enhancement tables for plugged.in proxy
-- These tables add stats, ratings, and curation features
-- Note: Uses 'proxy_' prefix to distinguish from upstream registry tables

-- Stats tracking for each server
CREATE TABLE IF NOT EXISTS proxy_server_stats (
  server_id TEXT PRIMARY KEY,
  installation_count INTEGER DEFAULT 0,
  rating DECIMAL(3,2) DEFAULT 0.00 CHECK (rating >= 0 AND rating <= 5),
  rating_count INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- User ratings and reviews
CREATE TABLE IF NOT EXISTS proxy_user_ratings (
  server_id TEXT NOT NULL,
  user_id VARCHAR(255) NOT NULL,
  rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
  comment TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (server_id, user_id)
);

-- Track individual installations
CREATE TABLE IF NOT EXISTS proxy_user_installations (
  server_id TEXT NOT NULL,
  user_id VARCHAR(255) NOT NULL,
  source VARCHAR(100),
  version VARCHAR(50),
  platform VARCHAR(50),
  installed_at TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (server_id, user_id)
);

-- Curated collections
CREATE TABLE IF NOT EXISTS collections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  owner_id VARCHAR(255),
  is_public BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Servers in collections
CREATE TABLE IF NOT EXISTS collection_servers (
  collection_id UUID REFERENCES collections(id) ON DELETE CASCADE,
  server_id UUID NOT NULL,
  added_at TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (collection_id, server_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_proxy_server_stats_rating ON proxy_server_stats(rating DESC, rating_count DESC);
CREATE INDEX IF NOT EXISTS idx_proxy_ratings_server ON proxy_user_ratings(server_id);
CREATE INDEX IF NOT EXISTS idx_proxy_ratings_user ON proxy_user_ratings(user_id);
CREATE INDEX IF NOT EXISTS idx_proxy_installations_server ON proxy_user_installations(server_id);
CREATE INDEX IF NOT EXISTS idx_proxy_installations_user ON proxy_user_installations(user_id);
CREATE INDEX IF NOT EXISTS idx_proxy_installations_date ON proxy_user_installations(installed_at DESC);
CREATE INDEX IF NOT EXISTS idx_collections_owner ON collections(owner_id);
CREATE INDEX IF NOT EXISTS idx_collections_public ON collections(is_public) WHERE is_public = true;