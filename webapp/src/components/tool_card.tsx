// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import styled from 'styled-components';
import {FormattedMessage} from 'react-intl';
import {ChevronDownIcon, ChevronRightIcon, DotsHorizontalIcon, CheckIcon, InformationOutlineIcon} from '@mattermost/compass-icons/components';

import {ToolCall, ToolCallStatus} from './llmbot_post/llmbot_post';

import LoadingSpinner from './assets/loading_spinner';
import IconTool from './assets/icon_tool';
import IconCheckCircle from './assets/icon_check_circle';
import DotMenu, {DropdownMenuItem} from './dot_menu';

// Styled components based on the Figma design
const ToolCallCard = styled.div`
    display: flex;
    flex-direction: column;
    padding: 12px 16px;
    border: 1px solid rgba(var(--center-channel-color-rgb), 0.08);
    border-radius: 4px;
    background: var(--center-channel-bg);
    box-shadow: 0px 1px 2px 0px rgba(0, 0, 0, 0.08);
    margin-bottom: 12px;
`;

const ToolCallHeader = styled.div`
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
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

const ToolIcon = styled(IconTool)`
    color: rgba(var(--center-channel-color-rgb), 0.64);
    min-width: 16px;
`;

const ToolName = styled.span`
    font-size: 11px;
    font-weight: 400;
    line-height: 16px;
    letter-spacing: 0.01em;
    color: rgba(var(--center-channel-color-rgb), 0.72);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex-grow: 1;
`;

const ToolCallDescription = styled.div`
    margin: 4px 0;
    font-size: 14px;
    color: rgba(var(--center-channel-color-rgb), 0.76);
`;

const ToolCallArguments = styled.pre`
    margin: 8px 0 12px;
    background: rgba(var(--center-channel-color-rgb), 0.04);
    padding: 12px;
    border-radius: 4px;
    overflow-x: auto;
    font-size: 12px;
    line-height: 1.4;
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

const SuccessIcon = styled(IconCheckCircle)`
    color: var(--online-indicator);
    min-width: 12px;
`;

const DecisionTag = styled.div<{approved?: boolean}>`
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin-left: auto;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    line-height: 16px;
    background: ${({approved}) => {
        if (approved === true) {
            return 'rgba(var(--center-channel-color-rgb), 0.08)';
        }
        if (approved === false) {
            return 'rgba(var(--error-text-color-rgb), 0.08)';
        }
        return 'transparent';
    }};
    color: ${({approved}) => {
        if (approved === true) {
            return 'var(--online-indicator)';
        }
        if (approved === false) {
            return 'var(--error-text)';
        }
        return 'inherit';
    }};
`;

const ButtonContainer = styled.div`
    display: flex;
    gap: 8px;
    margin-top: 16px;
`;

const ApproveButton = styled.button<{selected?: boolean, otherSelected?: boolean}>`
    background: ${({selected}) => (selected ? 'var(--online-indicator)' : 'var(--button-bg)')};
    color: var(--button-color);
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    line-height: 16px;
    cursor: pointer;
    flex: 1;
    opacity: ${({otherSelected}) => (otherSelected ? 0.5 : 1)};
    transition: opacity 0.15s ease-in-out;
    
    &:hover {
        background: ${({selected}) => {
        if (selected) {
            return 'rgba(var(--online-indicator-rgb), 0.88)';
        }
        return 'rgba(var(--button-bg-rgb), 0.88)';
    }};
        opacity: ${({otherSelected}) => (otherSelected ? 0.7 : 1)};
    }
    
    &:active {
        background: ${({selected}) => {
        if (selected) {
            return 'rgba(var(--online-indicator-rgb), 0.92)';
        }
        return 'rgba(var(--button-bg-rgb), 0.92)';
    }};
    }
`;

const RejectButton = styled.button<{selected?: boolean, otherSelected?: boolean}>`
    background: ${({selected}) => (selected ? 'var(--error-text)' : 'transparent')};
    color: ${({selected}) => (selected ? 'var(--button-color)' : 'var(--error-text)')};
    border: 1px solid var(--error-text);
    padding: 8px 16px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    line-height: 16px;
    cursor: pointer;
    flex: 1;
    opacity: ${({otherSelected}) => (otherSelected ? 0.5 : 1)};
    transition: opacity 0.15s ease-in-out;

    &:hover {
        background: ${({selected}) => {
        if (selected) {
            return 'rgba(var(--error-text-color-rgb), 0.88)';
        }
        return 'rgba(var(--error-text-color-rgb), 0.08)';
    }};
        color: ${({selected}) => (selected ? 'var(--button-color)' : 'var(--error-text)')};
        opacity: ${({otherSelected}) => (otherSelected ? 0.7 : 1)};
    }
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
    flex: 1;

    &:hover {
        background: rgba(var(--button-bg-rgb), 0.88);
    }

    &:active {
        background: rgba(var(--button-bg-rgb), 0.92);
    }
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

const ResultContainer = styled.pre`
    margin: 8px 0 0;
    padding: 12px;
    background: rgba(var(--center-channel-color-rgb), 0.04);
    border-radius: 4px;
    overflow-x: auto;
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.4;
`;

interface ToolCardProps {
    tool: ToolCall;
    isCollapsed: boolean;
    isProcessing: boolean;
    onToggleCollapse: () => void;
    onApprove?: () => void;
    onReject?: () => void;
    decision?: boolean | null; // true = approved, false = rejected, null = undecided
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
    decision,
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

    return (
        <ToolCallCard>
            <ToolCallHeader onClick={onToggleCollapse}>
                <StyledChevronIcon>
                    {effectivelyCollapsed ? <ChevronRightIcon size={16}/> : <ChevronDownIcon size={16}/>}
                </StyledChevronIcon>
                <ToolIcon/>
                <ToolName>{tool.name}</ToolName>

                {isPending && decision !== null && !isProcessing && !permissionsLoading && (
                    <DecisionTag approved={decision}>
                        {decision ? (
                            <FormattedMessage
                                id='ai.tool_call.will_approve'
                                defaultMessage='Will Approve'
                            />
                        ) : (
                            <FormattedMessage
                                id='ai.tool_call.will_reject'
                                defaultMessage='Will Reject'
                            />
                        )}
                    </DecisionTag>
                )}

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
                    <ToolCallDescription>{tool.description}</ToolCallDescription>
                    <ToolCallArguments>{JSON.stringify(tool.arguments, null, 2)}</ToolCallArguments>

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
                                <ApproveButton
                                    onClick={onApprove}
                                    selected={decision === true}
                                    otherSelected={decision === false}
                                    disabled={isProcessing}
                                >
                                    <FormattedMessage
                                        id='ai.tool_call.approve'
                                        defaultMessage='Approve'
                                    />
                                </ApproveButton>
                                <RejectButton
                                    onClick={onReject}
                                    selected={decision === false}
                                    otherSelected={decision === true}
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

                    {isAccepted && (
                        <StatusContainer>
                            <ProcessingSpinnerContainer>
                                <ProcessingSpinner/>
                            </ProcessingSpinnerContainer>
                            <FormattedMessage
                                id='ai.tool_call.status.processing'
                                defaultMessage='Processing...'
                            />
                        </StatusContainer>
                    )}

                    {isSuccess && (
                        <>
                            <StatusContainer>
                                <SuccessIcon/>
                                <FormattedMessage
                                    id='ai.tool_call.status.complete'
                                    defaultMessage='Complete'
                                />
                            </StatusContainer>
                            {tool.result && <ResultContainer>{tool.result}</ResultContainer>}
                        </>
                    )}

                    {isError && (
                        <>
                            <StatusContainer>
                                <span style={{color: 'var(--error-text)'}}>{'⚠️'}</span>
                                <FormattedMessage
                                    id='ai.tool_call.status.error'
                                    defaultMessage='Error'
                                />
                            </StatusContainer>
                            {tool.result && <ResultContainer>{tool.result}</ResultContainer>}
                        </>
                    )}

                    {isRejected && (
                        <StatusContainer>
                            <span>{'🚫'}</span>
                            <FormattedMessage
                                id='ai.tool_call.status.rejected'
                                defaultMessage='Rejected'
                            />
                        </StatusContainer>
                    )}
                </>
            )}
        </ToolCallCard>
    );
};

export default ToolCard;
