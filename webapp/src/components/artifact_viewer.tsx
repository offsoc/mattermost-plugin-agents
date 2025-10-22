// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import styled from 'styled-components';
import {useSelector} from 'react-redux';

import {GlobalState} from '@mattermost/types/store';
import {ChevronRightIcon, DownloadOutlineIcon, ContentCopyIcon} from '@mattermost/compass-icons/components';

import {Artifact, ArtifactMetadata} from './llmbot_post';

const ArtifactContainer = styled.div`
    margin-top: 12px;
    margin-bottom: 8px;
`;

const ArtifactBadge = styled.div`
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 12px;
    background: var(--center-channel-bg);
    border: 1px solid rgba(var(--center-channel-color-rgb), 0.16);
    border-radius: 4px;
    font-size: 14px;
    font-weight: 600;
    color: rgba(var(--center-channel-color-rgb), 0.88);
`;

const ArtifactBadgeLeft = styled.div`
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    user-select: none;
    flex: 1;
`;

const ChevronIcon = styled.div<{isExpanded: boolean}>`
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    transition: transform 0.2s ease;
    transform: ${(props) => (props.isExpanded ? 'rotate(90deg)' : 'rotate(0)')};

    svg {
        width: 14px;
        height: 14px;
    }
`;

const ArtifactList = styled.div<{isCollapsed: boolean}>`
    max-height: ${(props) => (props.isCollapsed ? '0' : '2000px')};
    overflow: hidden;
    transition: max-height 0.3s ease-in-out;
    opacity: ${(props) => (props.isCollapsed ? '0' : '1')};
    transition: opacity 0.2s ease-in-out, max-height 0.3s ease-in-out;
    margin-top: 0px;
`;

const ActionButton = styled.button`
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    font-size: 12px;
    font-weight: 600;
    color: rgba(var(--center-channel-color-rgb), 0.64);
    background: transparent;
    border: 1px solid rgba(var(--center-channel-color-rgb), 0.16);
    border-radius: 4px;
    cursor: pointer;

    &:hover {
        background: rgba(var(--center-channel-color-rgb), 0.08);
        color: rgba(var(--center-channel-color-rgb), 0.72);
    }
`;

const ArtifactCodeContainer = styled.div`
    position: relative;
    background: rgba(var(--center-channel-color-rgb), 0.02);
    border: 1px solid rgba(var(--center-channel-color-rgb), 0.08);
    border-radius: 4px;
`;

const ArtifactContentWrapper = styled.div`
    margin: 0;
    padding: 16px;

    /* Override Mattermost's code block styling to fit artifact container */
    pre {
        margin: 0;
        background: transparent !important;
        border: none !important;
        padding: 0 !important;
    }

    code {
        background: transparent !important;
        padding: 0 !important;
    }
`;

const LoadingCard = styled.div`
    background: var(--center-channel-bg);
    border: 1px solid rgba(var(--center-channel-color-rgb), 0.16);
    border-radius: 4px;
    padding: 16px;
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 12px;

    /* Pulse animation */
    animation: artifactPulse 2s ease-in-out infinite;

    @keyframes artifactPulse {
        0%, 100% {
            opacity: 1;
        }
        50% {
            opacity: 0.7;
        }
    }
`;

const LoadingContent = styled.div`
    display: flex;
    flex-direction: column;
    gap: 4px;
`;

const LoadingTitle = styled.div`
    font-size: 14px;
    font-weight: 600;
    color: rgba(var(--center-channel-color-rgb), 0.72);
`;

const LoadingStatus = styled.div`
    font-size: 12px;
    color: rgba(var(--center-channel-color-rgb), 0.56);
    font-weight: 600;
`;

interface Props {
    artifacts: Artifact[];
    generatingArtifacts?: ArtifactMetadata[];
    streamingContent?: string;
}

const StreamingCursor = styled.span`
    display: inline-block;
    width: 2px;
    height: 1em;
    background: rgba(var(--center-channel-color-rgb), 0.64);
    margin-left: 2px;
    animation: blink 1s step-end infinite;

    @keyframes blink {
        50% {
            opacity: 0;
        }
    }
`;

const ArtifactCode = ({content, language, showCursor}: {content: string; language?: string; showCursor?: boolean}) => {
    const siteURL = useSelector<GlobalState, string | undefined>((state) => state.entities.general.config.SiteURL);

    // @ts-ignore
    const {formatText, messageHtmlToComponent} = window.PostUtils;

    // Wrap code in markdown code fence for syntax highlighting
    const markdownCode = '```' + (language || '') + '\n' + content + '\n```';

    const markdownOptions = {
        singleline: false,
        mentionHighlight: false,
        atMentions: false,
        unsafeLinks: false,
        siteURL,
    };

    const messageHtmlToComponentOptions = {
        hasPluginTooltips: false,
        latex: false,
        inlinelatex: false,
    };

    const formattedCode = messageHtmlToComponent(
        formatText(markdownCode, markdownOptions),
        messageHtmlToComponentOptions,
    );

    return (
        <ArtifactContentWrapper>
            {formattedCode}
            {showCursor && <StreamingCursor/>}
        </ArtifactContentWrapper>
    );
};

export const ArtifactViewer = ({artifacts, generatingArtifacts, streamingContent}: Props) => {
    const [isExpanded, setIsExpanded] = useState(false);

    const totalCount = artifacts.length + (generatingArtifacts?.length || 0);

    // Auto-expand when streaming starts
    const isStreaming = Boolean(streamingContent);
    const shouldBeExpanded = isExpanded || isStreaming;

    if (totalCount === 0) {
        return null;
    }

    const handleCopy = (content: string) => {
        navigator.clipboard.writeText(content);
    };

    const handleDownload = (artifact: Artifact) => {
        const blob = new Blob([artifact.content], {type: 'text/plain'});
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;

        // Generate filename from title and use language as extension
        // AI is instructed to use proper file extensions (js, py, tsx, etc.)
        const filename = artifact.title.replace(/[^a-z0-9]/gi, '_').toLowerCase();
        const extension = artifact.language || 'txt';
        a.download = `${filename}.${extension}`;

        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    const getTypeIcon = (type: string) => {
        const iconStyle = {
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '20px',
            height: '20px',
            fontSize: '16px',
        };

        switch (type) {
        case 'code':
            return <span style={iconStyle}>{'</>'}</span>;
        case 'diagram':
            return <span style={iconStyle}>{'📊'}</span>;
        case 'document':
            return <span style={iconStyle}>{'📄'}</span>;
        default:
            return <span style={iconStyle}>{'</>'}</span>;
        }
    };

    // Get the first artifact (completed or generating) for the badge
    const firstArtifact = artifacts.length > 0 ? artifacts[0] : null;
    const firstGenerating = generatingArtifacts && generatingArtifacts.length > 0 ? generatingArtifacts[0] : null;

    const badgeTitle = firstArtifact?.title ||
        (firstGenerating?.title || (firstGenerating?.language ? `Untitled ${firstGenerating.language}` : 'Artifact'));

    return (
        <ArtifactContainer>
            <ArtifactBadge>
                <ArtifactBadgeLeft onClick={() => setIsExpanded(!isExpanded)}>
                    <ChevronIcon isExpanded={shouldBeExpanded}>
                        <ChevronRightIcon/>
                    </ChevronIcon>
                    {firstArtifact ? getTypeIcon(firstArtifact.type) : <span style={{fontSize: '16px'}}>{'</>'}</span>}
                    <span>{badgeTitle}</span>
                </ArtifactBadgeLeft>
                {firstArtifact && (
                    <div style={{display: 'flex', gap: '8px'}}>
                        <ActionButton onClick={() => handleCopy(firstArtifact.content)}>
                            <ContentCopyIcon size={16}/>
                        </ActionButton>
                        <ActionButton onClick={() => handleDownload(firstArtifact)}>
                            <DownloadOutlineIcon size={16}/>
                        </ActionButton>
                    </div>
                )}
            </ArtifactBadge>

            <ArtifactList isCollapsed={!shouldBeExpanded}>
                {/* Show streaming artifact code if generating and we have content */}
                {isStreaming && generatingArtifacts && generatingArtifacts.map((meta, index) => (
                    <ArtifactCodeContainer key={`streaming-${index}`}>
                        <ArtifactCode
                            content={streamingContent || ''}
                            language={meta.language}
                            showCursor={true}
                        />
                    </ArtifactCodeContainer>
                ))}

                {/* Show loading card only if generating but no streaming content yet */}
                {!isStreaming && generatingArtifacts && generatingArtifacts.map((meta, index) => {
                    const capitalizedLanguage = meta.language ?
                        meta.language.charAt(0).toUpperCase() + meta.language.slice(1) :
                        'Code';
                    return (
                        <LoadingCard key={`generating-${index}`}>
                            <span style={{fontSize: '20px', width: '20px', display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
                                {'</>'}
                            </span>
                            <LoadingContent>
                                <LoadingTitle>
                                    {meta.title || `Untitled ${capitalizedLanguage} Artifact`}
                                </LoadingTitle>
                                <LoadingStatus>
                                    {'💫 Generating '}{meta.language || 'code'}{'...'}
                                </LoadingStatus>
                            </LoadingContent>
                        </LoadingCard>
                    );
                })}

                {/* Show completed artifact code with syntax highlighting */}
                {artifacts.map((artifact, index) => (
                    <ArtifactCodeContainer key={index}>
                        <ArtifactCode
                            content={artifact.content}
                            language={artifact.language}
                            showCursor={false}
                        />
                    </ArtifactCodeContainer>
                ))}
            </ArtifactList>
        </ArtifactContainer>
    );
};
