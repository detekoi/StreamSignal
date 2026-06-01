export interface AnnouncementInput {
    streamTitle: string;
    streamURL: string;
    category: string;
    message: string;
    hashtags: string;
}

export interface PreviewItem {
    destinationID: string;
    destinationName: string;
    platform: string;
    previewLabel?: string;
    content: string;
    characterCount: number;
    validationState: string;
    validationNotes: string[];
}
