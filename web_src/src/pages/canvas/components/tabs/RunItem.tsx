import React, { JSX } from 'react';

interface RunItemProps {
  status: string;
  title: string;
  imageVersion: string;
  extraTags?: string;
  timestamp: string;
  isHightlighted?: boolean;
}

export const RunItem: React.FC<RunItemProps> = React.memo(({ 
  status, 
  title, 
  imageVersion, 
  extraTags, 
  timestamp, 
  isHightlighted = false 
}) => {
  const [isExpanded, setIsExpanded] = React.useState<boolean>(false);

  const toggleExpand = (): void => {
    setIsExpanded(!isExpanded);
  };

  const renderStatusIcon = (): JSX.Element | null => {
    switch (status.toLowerCase()) {
      case 'passed':
        return <span className="material-symbols-outlined fill green f1 mr1">check_circle</span>;
      case 'failed':
        return <span className="material-symbols-outlined fill red f1 mr1">cancel</span>;
      case 'queued':
        return <span className="material-symbols-outlined fill orange f1 mr1">queue</span>;
      case 'running':
        return <span className="br-pill bg-blue w-[22px] h-[22px] b--lightest-blue text-center mr2"><span className="white f4 job-log-working"></span></span>;
      default:
        return null;
    }
  };

  const getBackgroundClass = (): string => {
    if (!isHightlighted) return '';
    
    switch (status.toLowerCase()) {
      case 'passed':
        return 'bg-washed-green b--green';
      case 'failed':
        return 'bg-washed-red b--red';
      default:
        return 'bg-washed-blue b--indigo';
    }
  };

  return (
    <div className={`run-item flex items-start mv1 bg-white bb bl br br2 b--lightest-gray`}>
     <div className={`flex w-full items-start pa2 bt ${getBackgroundClass()}`}>
      <button 
          className="btn btn-outline btn-small py-0 px-0 leading-none mt1 mr1"
          onClick={toggleExpand}
          title={isExpanded ? "Hide details" : "Show details"}
        >
          <span className="material-symbols-outlined">{isExpanded ? 'arrow_drop_down' : 'arrow_right'}</span>
      </button>
      <div className='w-full'>
        <div className={`flex justify-between w-ful`}>
          <div className="flex items-center">
            {renderStatusIcon()}
            <img src={""} width={20} className="mx-1 hidden"/>
            <a href="#" className={`truncate b ${isExpanded ? "flex" : "flex"}`}>{title}</a>
          </div>
          <div className="flex items-center">
          <div className={`text-xs gray ml3-m ml0 mr3 tr ${isExpanded ? "hidden" : "inline-block"}`}>{timestamp}</div>
          <button className="btn gray text-lg px-1 py-0"><i className="material-symbols-outlined text-lg">more_vert</i></button>
          {status.toLowerCase() === 'queued' && <button className="btn btn-secondary btn-small text-sm"><i className="material-symbols-outlined text-sm">close</i></button>}
          </div>
        </div>
        <div className="w-full">
        
        <div className="flex items-start">
        <div className={`flex items-center pt1 ${isExpanded ? "hidden" : "flex"}`}>
              <span className="bg-black-10 black text-xs px-1 py-1 br2 mr2 leading-none ba b--black-20 code">image: {imageVersion}</span>
              {extraTags && (
                <span className="text-xs px-2 py-1 mr2">{extraTags}</span>
              )}
            </div>
         
          {isExpanded && (
            <div className="pt2">
              
                <div className="flex items-center text-sm">
                    <img src={""} width={16} className="mr1"/>
                    <span className="mr1 b hidden">Pipeline</span>
                    <a href="#" className="link dark-indigo underline">Semaphore project/Pipeline name</a>
                  </div>
                <div className="flex">
                  
                  <div className="flex items-center mt1">
                  <i className="hidden material-symbols-outlined mr1 text-sm">timer</i>
                    <div className="text-sm">
                      <div className="flex items-center">
                        <div className="flex items-center">
                          <i className="material-symbols-outlined text-sm gray mr1">nest_clock_farsight_analog</i> Jan 16, 2022 10:23:45
                          <div className='flex items-center ml3'><i className="material-symbols-outlined text-sm mid-gray mr1">hourglass_bottom</i> 00h 00m 25s</div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              <div className="flex justify-between mt-2">
                <div className='w-1/2'>
                  <div className="flex items-start"> 
                    <i className="material-symbols-outlined leading-1.2 mr1 f4">input</i>
                    <div className="text-sm">
                    <div className='mb1 ttu'>Inputs</div>
                      <div className="flex items-center code text-xs">
                        <div className='gray'>
                          <div>Code</div>
                          <div className='bg-black-05'>Image</div>
                          <div>Terraform</div>
                          <div>Something</div>
                        </div>
                        <div className=''>
                          <div className='pl2'>1045a77</div>
                          <div className='bg-black-05 pl2'>{imageVersion}</div>
                          <div className='pl2'>32.32</div>
                          <div className='pl2'>adsfasdf</div>
                        </div>
                      </div>
                    </div>
                  </div>
                
                </div>
        
                <div className={`w-1/2 bl br--black-075 pl3 ${status.toLowerCase() === 'passed' ? 'flex' :  'hidden'}`}>
                  <div className="flex items-start"> 
                    <i className="material-symbols-outlined mr1 text-sm">output</i>
                    <div className="text-sm">
                    <div className='mb1 ttu'>Outputs</div>
                      <div className="flex items-center code text-xs">
                        <div className='gray'>
                          <div>Code</div>
                          <div className='bg-black-05'>Image</div>
                          <div>Terraform</div>
                          <div>Something</div>
                        </div>
                        <div className=''>
                          <div className='pl2'>1045a77</div>
                          <div className='bg-black-05 pl2'>{imageVersion}</div>
                          <div className='pl2'>32.32</div>
                          <div className='pl2'>adsfasdf</div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
            </div>
              <div className='flex hidden'>
                <div className="flex items-start"> 
                    <i className="material-symbols-outlined mr1 text-sm">timer</i>
                    <div className="text-sm">
                    <div className='mb1 ttu'>Execution details</div>
                      <div className="flex items-center">
                        <div className='gray'>
                          <div>Date</div>
                          <div>Started</div>
                          <div>Finished</div>
                          <div>Duration</div>
                        </div>
                        <div className='ml2'>
                          <div>Jan 16, 2022</div>
                          <div>10:23:45</div>
                          <div>10:23:45</div>
                          <div>00h 00m 25s</div>
                        </div>
                      </div>
                    </div>
                  </div>
              </div>
            </div>
          )}
        </div>
        </div>
        </div>
      </div>
    </div>
  );
});