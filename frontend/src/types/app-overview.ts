export interface OverviewItem {
    title: string;
    description: string;
    status: string;
}

export interface AppOverview {
    productName: string;
    tagline: string;
    currentPhase: string;
    frontendStack: string;
    backendStack: string;
    dataStack: string;
    highlights: string[];
    milestones: OverviewItem[];
    architecture: OverviewItem[];
    nextActions: string[];
    acceptanceBars: string[];
}
