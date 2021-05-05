require_relative 'resp'
require_relative 'server'
require 'fileutils'
require 'tmpdir'

RSpec.describe 'File commands' do
    describe 'MKDIR' do
        before(:all) do
            $admin.put_array('MKDIR', 'world')
            @line = $admin.get_string
        end

        it 'returns OK' do
            expect(@line).to eq('OK')
        end

        it 'creates a folder' do
            newdir = File.join($dir, 'world')
            expect(Dir.exist? newdir).to be true
        end
    end
end