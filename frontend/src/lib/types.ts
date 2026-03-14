// TypeScript interfaces mirroring Go models in internal/models/.
// Property names use camelCase to match the JSON tags on Go structs.

// ---------------------------------------------------------------------------
// Enums / union types
// ---------------------------------------------------------------------------

export type QuestionType =
  | 'mcq'
  | 'essay'
  | 'fill_blank'
  | 'true_false'
  | 'short_answer'
  | 'match'
  | 'assertion_reasoning';

export type BloomLevel =
  | 'remember'
  | 'understand'
  | 'apply'
  | 'analyze'
  | 'evaluate'
  | 'create';

export type Difficulty = 'easy' | 'medium' | 'hard';

export type BookStatus = 'pending' | 'processing' | 'ready' | 'failed';

export type SourceType = 'url' | 'pdf' | 'search';

export type UserRole = 'student' | 'admin';

// ---------------------------------------------------------------------------
// Book
// ---------------------------------------------------------------------------

export interface TOCEntry {
  number: number;
  title: string;
  page?: number;
  depth: number;
}

export interface Book {
  id: string;
  title: string;
  author: string;
  isbn?: string;
  publisher?: string;
  language?: string;
  subject: string;
  gradeLevels: string[];
  educationSystem?: string;
  sourceType: string;
  sourceUrl?: string;
  pdfUrl?: string;
  coverImageUrl?: string;
  status: string;
  processingError?: string;
  toc?: TOCEntry[];
  metadata?: Record<string, string>;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

// ---------------------------------------------------------------------------
// Chapter
// ---------------------------------------------------------------------------

export interface Chapter {
  id: string;
  bookId: string;
  number: number;
  title: string;
  content?: string;
  summary?: string;
  topics?: string[];
  wordCount?: number;
  createdAt: string;
}

// ---------------------------------------------------------------------------
// Question
// ---------------------------------------------------------------------------

export interface Option {
  text: string;
  isCorrect: boolean;
}

export interface Enrichment {
  what: string;
  when: string;
  how: string;
  who: string;
}

export interface Question {
  id: string;
  bookId: string;
  chapterId: string;
  topic: string;
  questionText: string;
  questionType: QuestionType;
  difficulty: Difficulty;
  bloomLevel: BloomLevel;
  gradeLevel: string;
  examType?: string;
  options?: Option[];
  correctAnswer?: string;
  modelAnswer?: string;
  keyPoints?: string[];
  explanation?: string;
  enrichment: Enrichment;
  relatedQuestionIds?: string[];
  tags?: string[];
  createdAt: string;
}

// ---------------------------------------------------------------------------
// User & answers
// ---------------------------------------------------------------------------

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  gradeLevel?: string;
  educationSystem?: string;
  examPreparingFor?: string;
  avatarUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface UserAnswer {
  id: string;
  userId: string;
  questionId: string;
  answerText: string;
  isCorrect?: boolean;
  semanticScore?: number;
  keywordScore?: number;
  completenessScore?: number;
  overallScore?: number;
  feedback?: string;
  timeTaken?: number;
  createdAt: string;
}

// ---------------------------------------------------------------------------
// Admin
// ---------------------------------------------------------------------------

export interface AllowedSource {
  id: string;
  urlPattern: string;
  name: string;
  sourceType: string;
  enabled: boolean;
  addedBy?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

// ---------------------------------------------------------------------------
// Search (mirrors scraper.SearchResult, adapted for frontend use)
// ---------------------------------------------------------------------------

export interface SearchResult {
  title: string;
  author: string;
  isbn?: string;
  publisher?: string;
  language?: string;
  coverImageUrl?: string;
  description?: string;
  source: string;
  sourceUrl?: string;
}

// ---------------------------------------------------------------------------
// API response wrappers
// ---------------------------------------------------------------------------

export interface Pagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}

export interface UserStats {
  totalAnswered: number;
  averageScore: number;
  scoreByBloom: Record<string, number>;
  scoreByDifficulty: Record<string, number>;
  correctRate: number;
  averageTimeTaken: number;
}
