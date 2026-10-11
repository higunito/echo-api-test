// contracts/openapi.yaml の型

export type QuestStatus = "open" | "accepted" | "completed";

export type Adventurer = {
  id: string;
  name: string;
};

export type Quest = {
  id: string;
  title: string;
  description: string;
  reward: number;
  deadline: string | null;
  status: QuestStatus;
  is_expired: boolean;
  assignee: Adventurer | null;
  accepted_at: string | null;
  completed_at: string | null;
  created_at: string;
};

export type QuestPage = {
  items: Quest[];
  page: number;
  per_page: number;
  total_count: number;
  total_pages: number;
};

export type ErrorResponse = {
  error: { code: string; message: string };
};

export type CreateQuestInput = {
  title: string;
  description: string;
  reward: number;
  deadline: string | null;
};
