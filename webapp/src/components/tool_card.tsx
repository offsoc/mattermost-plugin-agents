// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useMemo, useState} from 'react';
import styled from 'styled-components';
import {FormattedMessage} from 'react-intl';
import {ChevronDownIcon, ChevronRightIcon, DotsHorizontalIcon, CheckIcon, AlertCircleOutlineIcon, CloseCircleOutlineIcon, InformationOutlineIcon} from '@mattermost/compass-icons/components';
import {useSelector} from 'react-redux';

import {GlobalState} from '@mattermost/types/store';

import manifest from '@/manifest';

import {ToolCall, ToolCallStatus} from './llmbot_post/llmbot_post';

import LoadingSpinner from './assets/loading_spinner';
import IconCheckCircle from './assets/icon_check_circle';
import DotMenu, {DropdownMenuItem} from './dot_menu';

// Styled components based on the Figma design
const ToolCallCard = styled.div`
    display: flex;
    flex-direction: column;
    margin-bottom: 4px;
    padding: 0;
    border: none;
    background: transparent;
    box-shadow: none;
`;

const ToolCallHeader = styled.div<{isCollapsed: boolean}>`
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: ${(props) => (props.isCollapsed ? '0' : '8px')};
    cursor: pointer;
    user-select: none;
`;

const StyledChevronIcon = styled.div`
    color: rgba(var(--center-channel-color-rgb), 0.56);
    min-width: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
`;

const StatusIcon = styled.div`
    color: rgba(var(--center-channel-color-rgb), 0.64);
    min-width: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
`;

const ToolName = styled.span`
    font-size: 14px;
    font-weight: 400;
    line-height: 20px;
    color: rgba(var(--center-channel-color-rgb), 0.75);
    flex-grow: 1;
`;

const ToolCallArguments = styled.div`
    margin: 0;
    padding-left: 13px;

    // Style code blocks rendered by Mattermost
    pre {
        margin: 0;
    }
`;

const StatusContainer = styled.div`
    display: flex;
    align-items: center;
    font-size: 11px;
    line-height: 16px;
    gap: 8px;
    color: rgba(var(--center-channel-color-rgb), 0.75);
    margin-top: 16px;
`;

const ProcessingSpinnerContainer = styled.div`
    display: flex;
    align-items: center;
    justify-content: center;
    width: 12px;
    height: 12px;
`;

const ProcessingSpinner = styled(LoadingSpinner)`
    width: 12px;
    height: 12px;
`;

const SmallSpinner = styled(LoadingSpinner)`
    width: 16px;
    height: 16px;
`;

const SmallSuccessIcon = styled(CheckIcon)`
    color: var(--online-indicator);
    min-width: 16px;
    width: 16px;
    height: 16px;
`;

const SmallErrorIcon = styled(AlertCircleOutlineIcon)`
    color: var(--error-text);
    min-width: 16px;
    width: 16px;
    height: 16px;
`;

const SmallRejectedIcon = styled(CloseCircleOutlineIcon)`
    color: var(--dnd-indicator);
    min-width: 16px;
    width: 16px;
    height: 16px;
`;

const ResponseSuccessIcon = styled(IconCheckCircle)`
    color: var(--online-indicator);
    min-width: 16px;
    width: 16px;
    height: 16px;
`;

const ResponseErrorIcon = styled(AlertCircleOutlineIcon)`
    color: var(--error-text);
    min-width: 16px;
    width: 16px;
    height: 16px;
`;

const ResponseRejectedIcon = styled(CloseCircleOutlineIcon)`
    color: var(--dnd-indicator);
    min-width: 16px;
    width: 16px;
    height: 16px;
`;

const DotMenuContainer = styled.div`
    margin-left: auto;
    display: flex;
    align-items: center;
`;

const MenuGroupTitle = styled.div`
    padding: 6px 20px;
    font-size: 12px;
    font-weight: 600;
    line-height: 16px;
    letter-spacing: 0.24px;
    text-transform: uppercase;
    color: rgba(var(--center-channel-color-rgb), 0.56);
    background: var(--center-channel-bg);
`;

const CheckIconContainer = styled.span`
    display: inline-flex;
    align-items: center;
    margin-left: auto;
    color: var(--button-bg);
    flex-shrink: 0;
`;

const MenuItemLabel = styled.span`
    display: flex;
    align-items: center;
    gap: 8px;
`;

const InfoIconWrapper = styled.span`
    position: relative;
    display: inline-flex;
    align-items: center;
`;

const InfoIcon = styled(InformationOutlineIcon)`
    color: rgba(var(--center-channel-color-rgb), 0.64);
    flex-shrink: 0;
    cursor: pointer;
`;

const TooltipContainer = styled.div`
    position: absolute;
    bottom: calc(100% + 8px);
    left: 50%;
    transform: translateX(-50%);
    z-index: 1000;
    pointer-events: none;
    opacity: 0;
    visibility: hidden;
    transition: opacity 0.2s ease, visibility 0.2s ease;

    ${InfoIconWrapper}:hover & {
        opacity: 1;
        visibility: visible;
    }
`;

const TooltipContent = styled.div`
    background: #1b1d22;
    border-radius: 4px;
    box-shadow: 0px 6px 14px 0px rgba(0, 0, 0, 0.12);
    padding: 6px 12px;
    font-family: 'Open Sans', sans-serif;
    font-weight: 400;
    font-size: 12px;
    line-height: 16px;
    color: white;
    min-width: 240px;
    max-width: 240px;
    word-wrap: break-word;
`;

const TooltipArrow = styled.div`
    position: absolute;
    bottom: -4px;
    left: 50%;
    transform: translateX(-50%);
    width: 0;
    height: 0;
    border-left: 4px solid transparent;
    border-right: 4px solid transparent;
    border-top: 4px solid #1b1d22;
`;

const PermissionMenuItem = styled(DropdownMenuItem)`
    display: flex !important;
    width: 100%;
    justify-content: space-between;
    align-items: center;
    gap: 32px;
`;

const ButtonContainer = styled.div`
    display: flex;
    gap: 8px;
    margin-top: 8px;
    padding-left: 42px;
`;

const AcceptAllButton = styled.button`
    background: var(--button-bg);
    color: var(--button-color);
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    line-height: 16px;
    cursor: pointer;

    &:hover {
        background: rgba(var(--button-bg-rgb), 0.88);
    }

    &:active {
        background: rgba(var(--button-bg-rgb), 0.92);
    }
`;

const AcceptButton = styled.button`
    background: rgba(var(--button-bg-rgb), 0.08);
    color: var(--button-bg);
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    line-height: 16px;
    cursor: pointer;

    &:hover {
        background: rgba(var(--button-bg-rgb), 0.12);
    }

    &:active {
        background: rgba(var(--button-bg-rgb), 0.16);
    }
`;

const RejectButton = styled.button`
    background: rgba(var(--button-bg-rgb), 0.08);
    color: var(--button-bg);
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    line-height: 16px;
    cursor: pointer;

    &:hover {
        background: rgba(var(--button-bg-rgb), 0.12);
    }

    &:active {
        background: rgba(var(--button-bg-rgb), 0.16);
    }
`;

const ResponseLabel = styled.div`
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 600;
    line-height: 20px;
    color: rgba(var(--center-channel-color-rgb), 0.75);
    padding-top: 8px;
    padding-bottom: 4px;
    padding-left: 13px;
`;

const ResultContainer = styled.div`
    margin: 0;
    padding-left: 13px;

    // Style code blocks rendered by Mattermost
    pre {
        margin: 0;
    }
`;

interface ToolCardProps {
    tool: ToolCall;
    isCollapsed: boolean;
    isProcessing: boolean;
    onToggleCollapse: () => void;
    onApprove?: () => void;
    onReject?: () => void;
    onAcceptAll?: () => void;
    onPermissionChange?: (permission: 'ask' | 'auto-approve') => void;
    autoApproved?: boolean;
    permissionsLoading?: boolean;
}

const ToolCard: React.FC<ToolCardProps> = ({
    tool,
    isCollapsed,
    isProcessing,
    onToggleCollapse,
    onApprove,
    onReject,
    onAcceptAll,
    onPermissionChange,
    autoApproved,
    permissionsLoading,
}) => {
    const [showTooltip, setShowTooltip] = useState(false);

    const isPending = tool.status === ToolCallStatus.Pending;
    const isAccepted = tool.status === ToolCallStatus.Accepted;
    const isSuccess = tool.status === ToolCallStatus.Success;
    const isError = tool.status === ToolCallStatus.Error;
    const isRejected = tool.status === ToolCallStatus.Rejected;

    // When permissions are loading for a pending tool, force collapsed state
    const effectivelyCollapsed = isCollapsed || (isPending && (permissionsLoading ?? false));

    // Convert underscores to spaces and capitalize first letter of each word
    // (e.g., "create_post" -> "Create Post")
    const displayName = tool.name.
        replace(/_/g, ' ').
        split(' ').
        map((word) => word.charAt(0).toUpperCase() + word.slice(1)).
        join(' ');

    const siteURL = useSelector<GlobalState, string | undefined>((state) => state.entities.general.config.SiteURL);
    const team = useSelector((state: GlobalState) => state.entities.teams.currentTeamId);
    const allowUnsafeLinks = useSelector<GlobalState, boolean>((state: any) => state['plugins-' + manifest.id]?.allowUnsafeLinks ?? false);

    // @ts-ignore
    const {formatText, messageHtmlToComponent} = window.PostUtils;

    const markdownOptions = {
        singleline: false,
        mentionHighlight: false,
        atMentions: false,
        team,
        unsafeLinks: !allowUnsafeLinks,
        minimumHashtagLength: 1000000000,
        siteURL,
    };

    const messageHtmlToComponentOptions = {
        hasPluginTooltips: false,
        latex: false,
        inlinelatex: false,
    };

    // Render arguments as JSON code block
    const argumentsMarkdown = `\`\`\`json\n${JSON.stringify(tool.arguments, null, 2)}\n\`\`\``;
    const renderedArguments = useMemo(() => {
        return messageHtmlToComponent(
            formatText(argumentsMarkdown, markdownOptions),
            messageHtmlToComponentOptions,
        );
    }, [tool.arguments]);

    // Render result as code block - try to detect if it's JSON
    const resultMarkdown = useMemo(() => {
        if (!tool.result) {
            return '';
        }

        // Try to parse as JSON to determine if we should use json syntax highlighting
        try {
            JSON.parse(tool.result);
            return `\`\`\`json\n${tool.result}\n\`\`\``;
        } catch {
            // Not JSON, use plain code block
            return `\`\`\`\n${tool.result}\n\`\`\``;
        }
    }, [tool.result]);

    const renderedResult = useMemo(() => {
        if (!tool.result || !resultMarkdown) {
            return null;
        }
        return messageHtmlToComponent(
            formatText(resultMarkdown, markdownOptions),
            messageHtmlToComponentOptions,
        );
    }, [resultMarkdown]);

    return (
        <ToolCallCard>
            <ToolCallHeader
                isCollapsed={effectivelyCollapsed}
                onClick={onToggleCollapse}
            >
                <StyledChevronIcon>
                    {effectivelyCollapsed ? <ChevronRightIcon size={16}/> : <ChevronDownIcon size={16}/>}
                </StyledChevronIcon>
                <StatusIcon>
                    {isPending && !isProcessing && !permissionsLoading && <SmallSpinner/>}
                    {isPending && permissionsLoading && <SmallSpinner/>}
                    {(isAccepted || (isPending && isProcessing)) && <SmallSpinner/>}
                    {isSuccess && <SmallSuccessIcon size={16}/>}
                    {isError && <SmallErrorIcon size={16}/>}
                    {isRejected && <SmallRejectedIcon size={16}/>}
                </StatusIcon>
                <ToolName>{displayName}</ToolName>

                {!isPending && onPermissionChange && (
                    <DotMenuContainer
                        onClick={(e) => {
                            e.stopPropagation();
                        }}
                    >
                        <DotMenu
                            icon={<DotsHorizontalIcon size={16}/>}
                            closeOnClick={true}
                        >
                            <MenuGroupTitle>
                                <FormattedMessage
                                    id='ai.tool_call.permission.menu_title'
                                    defaultMessage='On tool request'
                                />
                            </MenuGroupTitle>
                            <PermissionMenuItem
                                onClick={() => onPermissionChange('auto-approve')}
                            >
                                <MenuItemLabel>
                                    <FormattedMessage
                                        id='ai.tool_call.permission.auto_approve'
                                        defaultMessage='Allow everytime'
                                    />
                                    <InfoIconWrapper
                                        onMouseEnter={() => setShowTooltip(true)}
                                        onMouseLeave={() => setShowTooltip(false)}
                                    >
                                        <InfoIcon size={16}/>
                                        {showTooltip && (
                                            <TooltipContainer>
                                                <TooltipContent>
                                                    {'All allowed commands will be run automatically. Use at your own risk.'}
                                                </TooltipContent>
                                                <TooltipArrow/>
                                            </TooltipContainer>
                                        )}
                                    </InfoIconWrapper>
                                </MenuItemLabel>
                                {autoApproved && (
                                    <CheckIconContainer>
                                        <CheckIcon size={16}/>
                                    </CheckIconContainer>
                                )}
                            </PermissionMenuItem>
                            <PermissionMenuItem
                                onClick={() => onPermissionChange('ask')}
                            >
                                <FormattedMessage
                                    id='ai.tool_call.permission.ask'
                                    defaultMessage='Ask me everytime'
                                />
                                {!autoApproved && (
                                    <CheckIconContainer>
                                        <CheckIcon size={16}/>
                                    </CheckIconContainer>
                                )}
                            </PermissionMenuItem>
                        </DotMenu>
                    </DotMenuContainer>
                )}
            </ToolCallHeader>

            {!effectivelyCollapsed && (
                <>
                    <ToolCallArguments>{renderedArguments}</ToolCallArguments>

                    {(isSuccess || isError) && renderedResult && (
                        <>
                            <ResponseLabel>
                                {isSuccess && <ResponseSuccessIcon/>}
                                {isError && <ResponseErrorIcon/>}
                                <FormattedMessage
                                    id='ai.tool_call.response'
                                    defaultMessage='Response'
                                />
                            </ResponseLabel>
                            <ResultContainer>{renderedResult}</ResultContainer>
                        </>
                    )}

                    {isRejected && (
                        <StatusContainer>
                            <ResponseRejectedIcon/>
                            <FormattedMessage
                                id='ai.tool_call.status.rejected'
                                defaultMessage='Rejected'
                            />
                        </StatusContainer>
                    )}
                </>
            )}

            {isPending && !permissionsLoading && (
                isProcessing || autoApproved ? (
                    <StatusContainer>
                        <ProcessingSpinnerContainer>
                            <ProcessingSpinner/>
                        </ProcessingSpinnerContainer>
                        <FormattedMessage
                            id='ai.tool_call.processing'
                            defaultMessage='Processing...'
                        />
                    </StatusContainer>
                ) : (
                    <ButtonContainer>
                        {onAcceptAll && (
                            <AcceptAllButton
                                onClick={onAcceptAll}
                                disabled={isProcessing}
                            >
                                <FormattedMessage
                                    id='ai.tool_call.accept_all'
                                    defaultMessage='Accept all'
                                />
                            </AcceptAllButton>
                        )}
                        <AcceptButton
                            onClick={onApprove}
                            disabled={isProcessing}
                        >
                            <FormattedMessage
                                id='ai.tool_call.approve'
                                defaultMessage='Accept'
                            />
                        </AcceptButton>
                        <RejectButton
                            onClick={onReject}
                            disabled={isProcessing}
                        >
                            <FormattedMessage
                                id='ai.tool_call.reject'
                                defaultMessage='Reject'
                            />
                        </RejectButton>
                    </ButtonContainer>
                )
            )}
        </ToolCallCard>
    );
};

export default ToolCard;
