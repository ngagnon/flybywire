require 'open3'

RSpec.describe 'CLI' do
    it 'will not start with non-existent directory' do
        stdout, stderr, status = Open3.capture3('./bin/fly-server', '-notls', '-port', '7070', '/blah/blah/blah')
        expect(status.exitstatus).to eq(1)
        expect(stderr).to include('Root directory not found: /blah/blah/blah')
    end
end